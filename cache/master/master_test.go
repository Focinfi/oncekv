package master

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/meta"
	"github.com/Focinfi/oncekv/utils/mock"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

var (
	masterAddr = config.Config.CacheMasterAddr
	dbsKey     = config.Config.RaftNodesKey
)

func testNodes() nodesMap {
	return nodesMap{"127.0.0.1:55001": "127.0.0.1:55002"}
}

func TestNew(t *testing.T) {
	nodes := testNodes()
	b, err := json.Marshal(nodes)
	if err != nil {
		t.Fatal(err)
	}

	meta.Default.Put(cacheNodesKey, string(b))
	master := New(masterAddr)
	t.Logf("meta: %T, env=%v\n", master.meta, config.Config.Env)
	if !reflect.DeepEqual(master.nodesMap, nodes) {
		t.Errorf("can not init nodesMap, expect %v, got %v\n", nodes, master.nodesMap)
	}
}

func TestWatch(t *testing.T) {
	nodes := testNodes()
	master := New(masterAddr)

	// update meta
	dbs := []string{"127.0.0.1:55003", "127.0.0.1:55004"}
	b, err := json.Marshal(dbs)
	if err != nil {
		t.Fatal(err)
	}
	meta.Default.Put(dbsKey, string(b))

	// mock the watch period
	mock.DefaultWatchPeriod = time.Millisecond * 10
	// start watch
	go master.meta.WatchModify(master.nodesMapKey, func() { master.syncDBs() })

	// wait a second
	time.Sleep(time.Microsecond * 15)

	if !reflect.DeepEqual(master.nodesMap, nodes) {
		t.Errorf("can not watch nodesMap, expect %v, got %v\n", nodes, master.nodesMap)
	}
}

func TestHearbeat(t *testing.T) {
	nodes := testNodes()
	// init nodes
	nodes["127.0.0.1:55005"] = "127.0.0.1:55006"
	b, err := json.Marshal(nodes)
	if err != nil {
		t.Fatal(err)
	}
	meta.Default.Put(cacheNodesKey, string(b))

	// mock for first node
	httpPoster = mock.HTTPPosterCluster(map[string]mock.HTTPPoster{
		"127.0.0.1:55005": mock.MakeHTTPPoster("127.0.0.1:55005", "", nil, 0),
		"127.0.0.1:55001": mock.MakeHTTPPoster("127.0.0.1:55001", "", fmt.Errorf("broken"), 0),
	})
	master := New(masterAddr)

	// mock the heartbeat period
	defaultHeartbeatPeriod = time.Millisecond * 10
	// start heartbeat
	go master.heartbeat()

	// wait a second
	time.Sleep(time.Millisecond * 15)

	if _, ok := master.nodesMap["127.0.0.1:55005"]; !ok {
		t.Errorf("can not remove the alive node")
	}

	if _, ok := master.nodesMap["127.0.0.1:55001"]; ok {
		t.Errorf("should remove the dead node, current is: %v\n", master.nodesMap)
	}
}

func TestJoin(t *testing.T) {
	newNodeHTTP := "127.0.0.1:55007"
	newNodeInernal := "127.0.0.1:55008"
	// init nodes
	nodes := testNodes()
	b, err := json.Marshal(nodes)
	if err != nil {
		t.Fatal(err)
	}
	meta.Default.Put(cacheNodesKey, string(b))

	// new node server mock
	httpPoster = mock.MakeHTTPPoster(newNodeHTTP, "", nil, 0)

	master := New(masterAddr)
	go master.Start()

	param := joinParam{HTTPAddr: newNodeHTTP, NodeAddr: newNodeInernal}
	b, err = json.Marshal(param)
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("%s/join", urlutil.MakeURL(masterAddr))
	_, err = http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		t.Errorf("can not handle POST /join, err: %v\n", err)
	}

	// wait a moment
	time.Sleep(time.Millisecond * 10)

	if got, expect := master.nodesMap[urlutil.MakeURL(newNodeHTTP)], urlutil.MakeURL(newNodeInernal); got != expect {
		t.Errorf("can not join a node, expect %v, got: %v\n", expect, got)
	}
}
