package master

import (
	"fmt"
)

// StoreKey for kv store key
const StoreKey = "oncekv.nodes"

const httpAddrKey = "oncekv.nodes.http.adrr"

func httpAddrKeyOfRaftAddr(raftAddr string) string {
	return fmt.Sprintf("%s.%s", httpAddrKey, raftAddr)
}
