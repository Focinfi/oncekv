package mock

import (
	"fmt"
	"sync"

	"sort"

	"github.com/Focinfi/oncekv/log"
)

// Store mocks the db/node.Store
type Store struct {
	sync.RWMutex
	data   map[string]string
	leader string
	peers  []string
}

// NewStore returns a new Store
func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

// Open opens a store in a single mode or not
func (s *Store) Open(singleMode bool) error { return nil }

// Get returns the value for the given key.
func (s *Store) Get(key string) (string, error) {
	s.RLock()
	defer s.RUnlock()
	return s.data[key], nil
}

// Add adds key/value, via distributed consensus.
func (s *Store) Add(key, value string) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.data[key]; ok {
		return fmt.Errorf("duplicated")
	}

	s.data[key] = value
	log.DB.Infoln("mock Store:", s.data)
	return nil
}

// Join joins the node, reachable at addr, to the cluster.
func (s *Store) Join(addr string) error {
	s.Lock()
	defer s.Unlock()

	if len(s.peers) == 0 {
		peers := []string{s.leader, addr}
		sort.StringSlice(peers).Sort()
		s.peers = peers
	} else {
		s.peers = append(s.peers, addr)
	}

	return nil
}

// Peers returns the store peers
func (s *Store) Peers() ([]string, error) {
	s.RLock()
	defer s.RUnlock()

	return s.peers, nil
}

// Leader returns the leader address
func (s *Store) Leader() string { return s.leader }

// Stats return the stats as a map[string]string
func (s *Store) Stats() map[string]string { return map[string]string{} }

// SetLeader set the leader for testing
func (s *Store) SetLeader(leader string) {
	s.leader = leader
}
