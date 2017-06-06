package master

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"encoding/gob"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/meta"
	"github.com/Focinfi/oncekv/utils/consistenthash"
)

const (
	shardMasterServerStorageKey = "oncekv.shard.master.mapping"
	shardMasterMemberGroupKey   = "oncekv.shard.master.member.group"
)

func shardMasterMemberGroupKeyForID(gid int) string {
	return fmt.Sprintf("%s.%d", shardMasterMemberGroupKey, gid)
}

var (
	// ErrServiceUnavailable ShardMaster is broken
	ErrServiceUnavailable = errors.New("service is unavailable")
)

// Member for one server
type Member struct {
	Addr   string
	GID    int
	Extras map[string]string
}

// ShardMaster as shard master
type ShardMaster interface {
	Join(member Member) error
	Leave(member Member) error
	Move(shardIndex int, member Member) error
	Query(key string) (members [][]string, err error)
}

// shardIDToGroupIDs a sequence of the maping from shard index to group id as time goes by
type shardIDToGroupIDs []map[int]int

func init() {
	gob.Register(shardIDToGroupIDs{})
}

func (mapping shardIDToGroupIDs) groupIDs(shardID int) []int {
	gidMap := make(map[int]struct{})
	for i := len(mapping) - 1; i >= 0; i-- {
		if gid, ok := mapping[i][shardID]; ok {
			gidMap[gid] = struct{}{}
		}
	}

	gids := make([]int, len(gidMap))
	i := 0
	for k := range gidMap {
		gids[i] = k
	}

	return gids
}

type shardMasterServer struct {
	sync.RWMutex

	shardCount int
	keyMapping *consistenthash.Map
	shardIDToGroupIDs
}

// NewShardMasterServer allocates and returns a new shardMasterServer
func NewShardMasterServer() (ShardMaster, error) {
	server := &shardMasterServer{shardCount: config.Config.ShardCount}
	m := consistenthash.New(server.shardCount, nil)
	for i := 0; i < server.shardCount; i++ {
		m.Add(i)
	}
	server.keyMapping = m

	shardIDToGroupIDs, err := server.fetchShardIDToGroupIDs()
	if err != nil && err != config.ErrDataNotFound {
		return nil, fmt.Errorf("failed to fetch the shardIDToGroupIDs of shard id to group id, err:%v", err)
	}
	server.shardIDToGroupIDs = shardIDToGroupIDs

	return server, nil
}

func (server *shardMasterServer) fetchShardIDToGroupIDs() ([]map[int]int, error) {
	var mappings []map[int]int
	mappingsVal, err := meta.Default.Get(shardMasterServerStorageKey)
	if err != nil {
		return nil, err
	}

	if err := gob.NewDecoder(strings.NewReader(mappingsVal)).Decode(&mappings); err != nil {
		return nil, err
	}

	return mappings, nil
}

func (server *shardMasterServer) Join(member Member) error                 { return nil }
func (server *shardMasterServer) Leave(member Member) error                { return nil }
func (server *shardMasterServer) Move(shardIndex int, member Member) error { return nil }

func (server *shardMasterServer) Query(key string) ([][]string, error) {
	if server.shardIDToGroupIDs == nil {
		return nil, ErrServiceUnavailable
	}

	gids := server.shardIDToGroupIDs.groupIDs(server.keyMapping.Get(key))
	if len(gids) == 0 {
		return nil, ErrServiceUnavailable
	}

	groups := make([][]string, len(gids))

	for _, gid := range gids {
		key := shardMasterMemberGroupKeyForID(gid)
		membersVal, err := meta.Default.Get(key)

		if err != nil {
			return nil, fmt.Errorf("failed to get the value of '%s'", key)
		}

		members := []string{}
		if err := gob.NewDecoder(strings.NewReader(membersVal)).Decode(&members); err != nil {
			return nil, fmt.Errorf("data broken of key '%s'", key)
		}

		groups = append(groups, members)
	}

	return groups, nil
}
