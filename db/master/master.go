package master

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"net/http"

	"io/ioutil"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/meta"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

const (
	statsURLFormat = "%s/stats"
	logPerfix      = "db/master:"
)

var (
	raftNodesKey = config.Config().RaftNodesKey
)

// Master is the master of a raft group
type Master struct {
	sync.RWMutex
	stats map[string]map[string]string
	meta  meta.Meta
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
	for _, peer := range peers {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			stats, err := m.getNodeStats(url)
			if err != nil {
				log.DB.Error(err)
				m.Lock()
				defer m.Unlock()
				toRemove = append(toRemove, url)
				return
			}

			go m.updateStats(url, stats)
		}(peer)
	}

	wg.Wait()
	log.Biz.Infoln(logPerfix, "toRemove:", toRemove)
	if len(toRemove) > 0 {
		go m.removeNodes(toRemove)
	}
}

func (m *Master) updateStats(node string, stats map[string]string) {
	m.Lock()
	defer m.Unlock()

	m.stats[node] = stats
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

func (m *Master) getNodeStats(url string) (map[string]string, error) {
	url = fmt.Sprintf(statsURLFormat, urlutil.MakeURL(url))
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		stats := make(map[string]string)
		if err := json.Unmarshal(b, &stats); err != nil {
			return nil, err
		}

		return stats, nil
	}

	return nil, fmt.Errorf("%s %s returns %d\n", logPerfix, url, res.StatusCode)
}

// Peers returns the peers
func (m *Master) Peers() ([]string, error) {
	return m.fetchPeers()
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

	curPeers, err := m.fetchPeers()
	if err != nil {
		return err
	}

	log.Biz.Infoln(logPerfix, "UpdatePeers:", peers, curPeers)

	m.Lock()
	defer m.Unlock()
	if reflect.DeepEqual(peers, curPeers) {
		return nil
	}

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
var Default = &Master{
	stats: make(map[string]map[string]string),
}

func init() {
	Default.meta = meta.Default
}
