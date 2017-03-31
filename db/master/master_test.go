package master

import (
	"fmt"
	"testing"

	"time"

	"reflect"

	"github.com/Focinfi/oncekv/utils/mock"
)

func TestMaster(t *testing.T) {
	nodeOneHTTP := "127.0.0.1:55041"
	nodeOneRaft := "127.0.0.1:55042"
	nodeTwoHTTP := "127.0.0.1:55043"
	nodeTwoRaft := "127.0.0.1:55044"
	testHTTPPeers := []string{nodeOneHTTP, nodeTwoHTTP}
	getterMap := map[string]mock.HTTPGetter{
		nodeOneHTTP: mock.MakeHTTPGetter(nodeOneHTTP, "pong", nil, 0),
		nodeTwoHTTP: mock.MakeHTTPGetter(nodeTwoHTTP, "pong", nil, 0),
	}
	// mock the http getter
	httpGetter = mock.HTTPGetterCluster(getterMap)

	// register
	if err := Default.UpdatePeers(testHTTPPeers); err != nil {
		t.Fatal(err)
	}
	if err := Default.RegisterPeer(nodeOneRaft, nodeOneHTTP); err != nil {
		t.Fatal(err)
	}
	if err := Default.RegisterPeer(nodeTwoRaft, nodeTwoHTTP); err != nil {
		t.Fatal(err)
	}

	// mock the heartbeatPeriod
	heartbeatPeriod = time.Millisecond * 10
	go Default.Start()

	// wait heartbeat period
	time.Sleep(time.Millisecond * 15)

	if peers, err := Default.Peers(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(peers, testHTTPPeers) {
		t.Errorf("can not keep relationship for the peers, result: %v\n", peers)
	}

	// node two down
	getterMap[nodeTwoHTTP] = mock.MakeHTTPGetter(nodeTwoHTTP, "", fmt.Errorf("service error"), 0)
	// mock again
	httpGetter = mock.HTTPGetterCluster(getterMap)

	// wait a heartbeat period
	time.Sleep(time.Millisecond * 15)

	if peers, err := Default.Peers(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(peers, testHTTPPeers[:1]) {
		t.Errorf("can not keep relationship for the peers, result: %v\n", peers)
	}
}
