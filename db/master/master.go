package master

import (
	"encoding/json"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/meta"
)

var (
	raftNodesKey = config.Config().RaftNodesKey
)

// Master is the master of a raft group
type Master struct {
	meta meta.Meta
}

// Peers returns the peers
func (m *Master) Peers() ([]string, error) {
	peers := []string{}
	val, err := m.meta.Get(raftNodesKey)
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
	b, err := json.Marshal(peers)
	if err != nil {
		return err
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
var Default = &Master{}

func init() {
	Default.meta = meta.Default
}
