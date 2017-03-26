package master

import (
	"fmt"

	"github.com/Focinfi/oncekv/config"
)

var (
	httpAddrKey = config.Config.RaftKey
)

func httpAddrKeyOfRaftAddr(raftAddr string) string {
	return fmt.Sprintf("%s.%s", httpAddrKey, raftAddr)
}
