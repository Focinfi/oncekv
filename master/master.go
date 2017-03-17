package master

import "encoding/json"

// Master is the master of a raft group
type Master struct {
	meta kv
}

// Peers returns the peers
func (m *Master) Peers() ([]string, error) {
	peers := []string{}
	val, err := m.meta.Get(StoreKey)
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

	return m.meta.Set(StoreKey, string(b))
}

// RegisterPeer register peer
func (m *Master) RegisterPeer(raftAddr, httpAddr string) error {
	_, err := m.PeerHTTPAddr(raftAddr)
	if err != nil {
		return err
	}

	return m.meta.Set(httpAddrKeyOfRaftAddr(raftAddr), httpAddr)
}

// PeerHTTPAddr get the httpAddr for the raft
func (m *Master) PeerHTTPAddr(raftAddr string) (string, error) {
	return m.meta.Get(httpAddrKeyOfRaftAddr(raftAddr))
}

// Default for the default master
var Default = &Master{}

func init() {
	kv, err := newEtcdKV()
	if err != nil {
		panic(err)
	}

	Default.meta = kv
}
