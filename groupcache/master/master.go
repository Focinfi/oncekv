package master

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/master"
	"github.com/Focinfi/oncekv/utils/urlutil"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
)

const (
	defaultHeartbeatPeriod = time.Second * 1
	defaultKey             = "oncekv.groupcache.master"
	defaultEtcdEndpoint    = "localhost:2379"
	defaultAddr            = ":5550"
	jsonHTTPHeader         = "application/json"
	heartbeatURLFormat     = "%s/meta"
	logPrefix              = "groupcache/master"
)

// nodesMap is pairs of httpAddr/nodeAddr
type nodesMap map[string]string

func (p nodesMap) httpAddrs() []string {
	addrs := make([]string, len(p))
	i := 0
	for k := range p {
		addrs[i] = k
		i++
	}

	return addrs
}

func (p nodesMap) nodeAddrs() []string {
	addrs := make([]string, len(p))
	i := 0
	for k := range p {
		addrs[i] = p[k]
		i++
	}

	return addrs
}

type peerParam struct {
	Peers []string `json:"peers"`
	DBs   []string `json:"dbs"`
}

// Master for a group of caching nodes
type Master struct {
	// runtime data
	sync.RWMutex
	nodesMap nodesMap
	dbs      []string

	// http server
	server *gin.Engine
	addr   string

	// databse store
	store       *clientv3.Client
	nodesMapKey string
}

// New returns a new Master with the addr
func New(addr string) *Master {
	store, err := clientv3.New(
		clientv3.Config{
			Endpoints: []string{defaultEtcdEndpoint},
		},
	)

	if err != nil {
		panic(err)
	}

	master := &Master{
		addr:        addr,
		store:       store,
		nodesMapKey: defaultKey,
	}

	nodesMap, err := master.fetchNodesMap()
	if err != nil {
		panic(err)
	}

	log.Biz.Infoln(logPrefix, "Nodes: ", nodesMap)

	master.nodesMap = nodesMap
	master.server = newServer(master)
	return master
}

// Start starts the master listening on addr
func (m *Master) Start() {
	go m.watchDBs()
	go m.heartbeat()
	log.Biz.Fatal(m.server.Run(m.addr))
}

func newServer(m *Master) *gin.Engine {
	server := gin.Default()
	server.POST("/join", m.handleJoinNode)
	return server
}

func (m *Master) setNodesMap(peers nodesMap) {
	m.Lock()
	defer m.Unlock()

	m.nodesMap = peers
}

func (m *Master) handleJoinNode(ctx *gin.Context) {
	var params = &struct {
		HTTPAddr string `json:"httpAddr"`
		NodeAddr string `json:"nodeAddr"`
	}{}

	err := ctx.BindJSON(params)
	if err != nil || params.HTTPAddr == "" || params.NodeAddr == "" {
		ctx.JSON(http.StatusBadRequest, nil)
		return
	}

	m.Lock()
	m.nodesMap[urlutil.MakeURL(params.HTTPAddr)] = urlutil.MakeURL(params.NodeAddr)
	if err := m.updateNodesMap(m.nodesMap); err != nil {
		ctx.JSON(http.StatusInternalServerError, nil)
		m.Unlock()
		return
	}

	ctx.JSON(http.StatusOK, peerParam{Peers: m.nodesMap.httpAddrs(), DBs: m.dbs})
	m.Unlock()
}

func (m *Master) fetchNodesMap() (nodesMap, error) {
	nodes := nodesMap{}

	val, err := m.store.Get(context.TODO(), m.nodesMapKey)
	if err != nil || len(val.Kvs) == 0 {
		return nil, err
	}

	if err := json.Unmarshal(val.Kvs[0].Value, &nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (m *Master) updateNodesMap(peers nodesMap) error {
	b, err := json.Marshal(peers)
	if err != nil {
		return err
	}

	_, err = m.store.Put(context.TODO(), m.nodesMapKey, string(b))
	if err != nil {
		return err
	}

	return nil
}

// heartbeat for check the nodes health periodicly
func (m *Master) heartbeat() {
	ticker := time.NewTicker(defaultHeartbeatPeriod)
	for {
		<-ticker.C
		m.RLock()
		nodesMap := m.nodesMap
		m.RUnlock()

		nodePeers := nodesMap.nodeAddrs()
		for _, nodeURL := range nodesMap.httpAddrs() {
			go func(node string) {
				err := m.sendPeers(node, nodePeers)
				if err != nil {
					log.Internal.Errorln(logPrefix, "node error:", err)
					m.removeNode(node)
				}
			}(nodeURL)
		}
	}
}

func (m *Master) sendPeers(node string, nodes []string) error {
	if err := m.syncDBs(); err != nil {
		return err
	}

	params := peerParam{Peers: nodes, DBs: m.dbs}

	b, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf(heartbeatURLFormat, urlutil.MakeURL(node)), jsonHTTPHeader, bytes.NewReader(b))

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("failed to send peers")
	}

	return err
}

func (m *Master) removeNode(node string) {
	m.Lock()
	if _, ok := m.nodesMap[node]; !ok {
		m.Unlock()
		return
	}

	delete(m.nodesMap, node)

	if err := m.updateNodesMap(m.nodesMap); err != nil {
		log.DB.Errorln(logPrefix, "database error:", err)
	}

	log.DB.Errorln(logPrefix, node, "remoted")
	m.Unlock()
}

func (m *Master) watchDBs() {
	ch := m.store.Watch(context.TODO(), master.StoreKey)

	for {
		resp := <-ch
		log.DB.Infoln(logPrefix, "watchDBs:", string(resp.Events[0].Kv.Value))
		if resp.Canceled {
			ch = m.store.Watch(context.TODO(), master.StoreKey)
			continue
		}

		if err := resp.Err(); err != nil {
			log.DB.Infoln(logPrefix, "failed to watch dbs, err: ", err)
		}

		for _, event := range resp.Events {
			if event.IsModify() {
				m.syncDBs()
			}
		}
	}
}

func (m *Master) syncDBs() error {
	dbs, err := master.Default.Peers()
	if err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()
	m.dbs = dbs

	return nil
}

var defaultMaster = New(defaultAddr)

// Peers returns peers of defaultKey
func Peers() ([]string, error) {
	peers, err := defaultMaster.fetchNodesMap()
	if err != nil {
		return nil, err
	}

	return peers.httpAddrs(), nil
}
