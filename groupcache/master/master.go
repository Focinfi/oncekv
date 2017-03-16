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
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
)

const defaultHeartbeatPeriod = time.Second * 1
const defaultKey = "oncekv.groupcache.master"
const defaultEtcdEndpoint = "localhost:2379"

// Master for a group of caching nodes
type Master struct {
	sync.RWMutex
	addr   string
	server *gin.Engine
	store  *clientv3.Client

	nodeKey string
	nodeMap map[string]bool
	dbs     []string
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

	nodes, err := master.fetchNodes()
	if err != nil {
		panic(err)
	}
	fmt.Println("Nodes: ", nodes)
	master.setNodes(nodes)

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

func (m *Master) nodes() []string {
	nodes := make([]string, len(m.nodeMap))
	i := 0
	for k := range m.nodeMap {
		nodes[i] = k
		i++
	}

	return nodes
}

func (m *Master) setNodes(nodes []string) {
	newNodeMap := make(map[string]bool)
	for _, node := range nodes {
		newNodeMap[node] = true
	}

	m.Lock()
	defer m.Unlock()

	m.nodeMap = newNodeMap
}

func (m *Master) handleGetNodes(ctx *gin.Context) {
	path := ctx.Request.URL.Path
	m.RLock()
	if _, ok := m.nodeMap[path]; !ok {
		m.RUnlock()
		ctx.JSON(http.StatusMethodNotAllowed, nil)
		return
	}

	ctx.JSON(http.StatusOK, m.nodes())
	m.RUnlock()
}

func (m *Master) handleJoinNode(ctx *gin.Context) {
	var params = &struct {
		Addr string `json:"addr"`
	}{}

	err := ctx.BindJSON(params)
	if err != nil || params.Addr == "" {
		ctx.JSON(http.StatusBadRequest, nil)
		return
	}

	m.Lock()
	if _, ok := m.nodeMap[params.Addr]; ok {
		m.Unlock()
		ctx.JSON(http.StatusBadRequest, nil)
		return
	}

	m.nodeMap[params.Addr] = true
	nodes := m.nodes()
	if err := m.updateNodes(nodes); err != nil {
		ctx.JSON(http.StatusInternalServerError, nil)
		m.Unlock()
		return
	}

	m.Unlock()
	ctx.JSON(http.StatusOK, peerParam{Peers: nodes, DBs: m.dbs})
}

func (m *Master) fetchNodes() ([]string, error) {
	nodes := []string{}

	val, err := m.store.Get(context.TODO(), m.nodeKey)
	if err != nil || len(val.Kvs) == 0 {
		return nodes, err
	}

	if err := json.Unmarshal(val.Kvs[0].Value, &nodes); err != nil {
		return nodes, err
	}

	return nodes, nil
}

func (m *Master) updateNodes(nodes []string) error {
	b, err := json.Marshal(nodes)
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
		nodes := m.nodes()
		for k := range m.nodeMap {
			go func(node string) {
				err := m.sendPeers(node, nodes)
				if err != nil {
					fmt.Println("ERR: ", err.Error())
					m.removeNode(node)
				}
			}(k)
		}
		m.RUnlock()
	}
}

func (m *Master) sendPeers(node string, nodes []string) error {
	params := peerParam{Peers: nodes, DBs: m.dbs}

	b, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("http://%s/meta", node), "application/json", bytes.NewReader(b))

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
	if _, ok := m.nodeMap[node]; !ok {
		m.Unlock()
		return
	}

	delete(m.nodeMap, node)

	if err := m.updateNodes(m.nodes()); err != nil {
		log.Println("etcd error: " + err.Error())
	}

	log.Println(node, " remoted")
	m.Unlock()
}

func (m *Master) watchDBs() {
	ch := m.store.Watch(context.TODO(), master.StoreKey)

	for {
		resp := <-ch
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
	res, err := m.store.Get(context.TODO(), master.StoreKey)
	if err != nil {
		return err
	}

	if len(res.Kvs) == 0 {
		return errors.New("dbs data lost")
	}

	dbs := []string{}
	if err := json.Unmarshal(res.Kvs[0].Value, &dbs); err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()

	m.dbs = dbs

	return nil
}
