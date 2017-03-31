package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

import "github.com/Focinfi/oncekv/utils/mock"

var (
	httpAddr       = "127.0.0.1:55441"
	groupcacheAddr = "127.0.0.1:55442"
	peers          = []string{"127.0.0.1:50001", "127.0.0.1:50002"}
	dbs            = []string{"127.0.0.1:50003"}
	masterAddr     = config.Config.CacheMasterAddr
)

func mockMaster(t *testing.T, peers, dbs []string) {
	b, err := json.Marshal(masterParam{
		Peers: peers,
		DBs:   dbs,
	})
	if err != nil {
		t.Fatal(err)
	}

	// mock the poster
	httpPoster = mock.MakeHTTPPoster(masterAddr, string(b), nil, 0)
}

func TestJoinAndMeta(t *testing.T) {
	mockMaster(t, peers, dbs)
	n := New(httpAddr, groupcacheAddr, masterAddr)
	go n.Start()
	time.Sleep(time.Millisecond)

	if !reflect.DeepEqual(n.peers, peers) {
		t.Errorf("failed to init peers, expect: %v, got: %v\n", peers, n.peers)
	}

	if !reflect.DeepEqual(n.dbs, dbs) {
		t.Errorf("failed to init peers, expect: %v, got: %v\n", dbs, n.dbs)
	}

	newPeers := append(peers, "127.0.0.1:50006")
	newDBs := append(dbs, "127.0.0.1:50004")
	b, err := json.Marshal(masterParam{
		Peers: newPeers,
		DBs:   newDBs,
	})
	if err != nil {
		panic(err)
	}

	url := fmt.Sprintf("%s/meta", urlutil.MakeURL(httpAddr))
	res, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("n can not handle POST /meta, response code is %d\n", res.StatusCode)
	}

	if !reflect.DeepEqual(n.peers, newPeers) {
		t.Errorf("failed to update peers after /meta, expect: %v, got: %v\n", newPeers, n.peers)
	}

	if !reflect.DeepEqual(n.dbs, newDBs) {
		t.Errorf("failed to update peers after /meta, expect: %v, got: %v\n", newDBs, n.dbs)
	}

	// mock with getters
	getters := map[string]mock.HTTPGetter{}
	getters[newDBs[0]] = mock.MakeHTTPGetter(newDBs[0], `{"key":"foo","value":"bar"}`, nil, 0)
	getters[newDBs[1]] = mock.MakeHTTPGetter(newDBs[1], "{}", fmt.Errorf("i am not a leader"), 0)
	httpGetter = mock.HTTPGetterCluster(getters)

	url = fmt.Sprintf("%s/key/foo", urlutil.MakeURL(httpAddr))
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(fmt.Sprintf("GET /key/:key is not available, response code: %d\n", resp.StatusCode))
	}

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	resMap := map[string]string{}
	if err := json.Unmarshal(b, &resMap); err != nil {
		t.Fatal(err)
	}

	if value := resMap["value"]; value != "bar" {
		t.Errorf("failed to response the value, expect bar, got %s\n", value)
	}

	if n.fastDB != dbs[0] {
		t.Errorf("failed to set fastDB, expect: %s, got: %v\n", dbs[0], n.fastDB)
	}
}
