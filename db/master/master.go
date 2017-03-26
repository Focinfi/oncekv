package master

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/meta"
	"github.com/Focinfi/oncekv/utils/mock"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

const (
	statsURLFormat = "%s/stats"
	logPerfix      = "db/master:"
)

var (
	raftNodesKey = config.Config.RaftNodesKey
)

// Master is the master of a raft group
type Master struct {
	meta   meta.Meta
	getter mock.HTTPGetter
}

// Peers returns the peers
func (m *Master) Peers() ([]string, error) {
	return m.fetchPeers()
}

// Start starts manage peers
func (m *Master) Start() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			m.heartbeat()
		}
	}
}

func (m *Master) heartbeat() {
	peers, err := m.fetchPeers()
	if err != nil {
		log.DB.Error(err)
		return
	}

	log.Biz.Infoln(peers)
	toRemove := []string{}
	var wg sync.WaitGroup
	var mux sync.Mutex

	for _, peer := range peers {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			_, err := m.getter.Get(urlutil.MakeURL(url))
			if err != nil {
				log.DB.Error(err)
				mux.Lock()
				defer mux.Unlock()
				toRemove = append(toRemove, url)
				return
			}
		}(peer)
	}

	wg.Wait()
	log.Biz.Infoln(logPerfix, "toRemove:", toRemove)
	if len(toRemove) > 0 {
		go m.removeNodes(toRemove)
	}
}

func (m *Master) removeNodes(nodes []string) {
	if len(nodes) == 0 {
		return
	}

	toRemove := map[string]bool{}
	for _, node := range nodes {
		toRemove[node] = true
	}

	curNodes, err := m.fetchPeers()
	if err != nil {
		log.DB.Error(err)
		return
	}
	newPeers := []string{}

	var changed bool
	for _, peer := range curNodes {
		if _, ok := toRemove[peer]; !ok {
			newPeers = append(newPeers, peer)
		} else {
			changed = true
		}
	}

	if changed {
		m.UpdatePeers(newPeers)
	}
}

func (m *Master) fetchPeers() ([]string, error) {
	peers := []string{}
	val, err := m.meta.Get(raftNodesKey)
	if err == config.ErrDataNotFound {
		return peers, nil
	}

	if err != nil {
		return peers, err
	}

	if err := json.Unmarshal([]byte(val), &peers); err != nil {
		return peers, err
	}

	return peers, nil
}

// UpdatePeers updatePeers into store
func (m *Master) UpdatePeers(peers []string) error {
	sort.StringSlice(peers).Sort()
	b, err := json.Marshal(peers)
	if err != nil {
		return err
	}

	log.DB.Info("To Update perrs:", peers)
	return m.meta.Put(raftNodesKey, string(b))
}

// RegisterPeer register peer
func (m *Master) RegisterPeer(raftAddr, httpAddr string) error {
	return m.meta.Put(httpAddrKeyOfRaftAddr(raftAddr), httpAddr)
}

// PeerHTTPAddr get the httpAddr for the raft
func (m *Master) PeerHTTPAddr(raftAddr string) (string, error) {
	return m.meta.Get(httpAddrKeyOfRaftAddr(raftAddr))
}

// Default for the default master
var Default *Master

func init() {
	Default = &Master{
		meta:   meta.Default,
		getter: mock.HTTPGetterFunc(http.Get),
	}
}
