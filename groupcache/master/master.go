package master

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/master"
	"github.com/Focinfi/oncekv/utils/urlutil"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
)

const defaultHeartbeatPeriod = time.Second * 1
const defaultKey = "oncekv.groupcache.master"
const defaultEtcdEndpoint = "localhost:2379"
const defaultAddr = ":5550"

type nodesMap map[string]string // httpAddr/nodeAddr

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

// Master for a group of caching nodes
type Master struct {
	sync.RWMutex
	addr   string
	server *gin.Engine
	store  *clientv3.Client

	nodeKey  string
	nodesMap nodesMap
	dbs      []string
}

type peerParam struct {
	Peers []string `json:"peers"`
	DBs   []string `json:"dbs"`
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
		addr:    addr,
		store:   store,
		nodeKey: defaultKey,
	}

	nodesMap, err := master.fetchNodesMap()
	if err != nil {
		panic(err)
	}
	fmt.Println("Nodes: ", nodesMap)
	master.setNodesMap(nodesMap)

	master.server = newServer(master)

	return master
}

// Start starts the master listening on addr
func (m *Master) Start() {
	go m.watchDBs()
	go m.heartbeat()
	log.Fatal(m.server.Run(m.addr))
}

func newServer(m *Master) *gin.Engine {
	server := gin.Default()
	server.POST("/join", m.handleJoinNode)
	server.GET("/nodes", m.handleGetNodes)
	return server
}

func (m *Master) setNodesMap(peers nodesMap) {
	m.Lock()
	defer m.Unlock()

	m.nodesMap = peers
}

func (m *Master) handleGetNodes(ctx *gin.Context) {
	path := ctx.Request.URL.Path
	m.RLock()
	if _, ok := m.nodesMap[path]; !ok {
		m.RUnlock()
		ctx.JSON(http.StatusMethodNotAllowed, nil)
		return
	}

	ctx.JSON(http.StatusOK, m.nodesMap.httpAddrs())
	m.RUnlock()
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

	val, err := m.store.Get(context.TODO(), m.nodeKey)
	if err != nil || len(val.Kvs) == 0 {
		return nodes, err
	}

	if err := json.Unmarshal(val.Kvs[0].Value, &nodes); err != nil {
		return nodes, err
	}

	return nodes, nil
}

func (m *Master) updateNodesMap(peers nodesMap) error {
	b, err := json.Marshal(peers)
	if err != nil {
		return err
	}

	_, err = m.store.Put(context.TODO(), m.nodeKey, string(b))
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
		httpAddrs := nodesMap.httpAddrs()
		nodePeers := nodesMap.nodeAddrs()
		for _, nodeURL := range httpAddrs {
			go func(node string) {
				err := m.sendPeers(node, nodePeers)
				if err != nil {
					fmt.Println("ERR: ", err.Error())
					m.removeNode(node)
				}
			}(nodeURL)
		}
		m.RUnlock()
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

	res, err := http.Post(fmt.Sprintf("%s/meta", urlutil.MakeURL(node)), "application/json", bytes.NewReader(b))

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("failed to send peers")
	}

	return err
}

func (m *Master) removeNode(peer string) {
	m.Lock()
	if _, ok := m.nodesMap[peer]; !ok {
		m.Unlock()
		return
	}

	delete(m.nodesMap, peer)

	if err := m.updateNodesMap(m.nodesMap); err != nil {
		log.Println("etcd error: " + err.Error())
	}

	log.Println(peer, " remoted")
	m.Unlock()
}

func (m *Master) watchDBs() {
	ch := m.store.Watch(context.TODO(), master.StoreKey)

	for {
		resp := <-ch
		log.Printf("WatchDBs: %v\n", string(resp.Events[0].Kv.Value))
		if resp.Canceled {
			ch = m.store.Watch(context.TODO(), master.StoreKey)
			continue
		}

		if err := resp.Err(); err != nil {
			log.Println("failed to watch dbs, err: ", err.Error())
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
