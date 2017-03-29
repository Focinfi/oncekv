package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/db/master"
	"github.com/Focinfi/oncekv/utils/mock"
	"github.com/Focinfi/oncekv/utils/urlutil"
)

var (
	testHTTPAddr   = "127.0.0.1:55501"
	testRaftAddr   = "127.0.0.1:55502"
	jsonHTTPHeader = "application/json"
)

func TestNode(t *testing.T) {
	node := New(testHTTPAddr, testRaftAddr, "")
	store := mock.NewStore()
	store.SetLeader(testRaftAddr)
	node.store = store
	go node.Start()

	// POST /key
	param := map[string]string{"key": "foo", "value": "bar"}
	b, err := json.Marshal(param)
	if err != nil {
		t.Fatal(err)
	}
	postURL := fmt.Sprintf("%s/key", urlutil.MakeURL(testHTTPAddr))
	resp, err := http.Post(postURL, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to add foo/bar, status code: %d\n", resp.StatusCode)
	}
	resp.Body.Close()

	// GET /i/key/:key
	getURL := fmt.Sprintf("%s/i/key/foo", urlutil.MakeURL(testHTTPAddr))
	resp, err = http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get the value of 'foo', status code: %d\n", resp.StatusCode)
	}

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	respKV := &kvResp{}
	if err := json.Unmarshal(b, respKV); err != nil {
		t.Fatalf("response of get has wrong format, err: %v\n", err)
	}

	if respKV.Key != "foo" || respKV.Value != "bar" {
		t.Errorf("failed to get 'foo', expect: foo/bar, got: %v\n", respKV)
	}

	// new node try to join
	newNodeHTTP := "127.0.0.1:55503"
	newRaftNode := "127.0.0.1:55504"

	newNode := New(newNodeHTTP, newRaftNode, "")
	newNode.store = mock.NewStore()
	go newNode.Start()
	time.Sleep(time.Millisecond * 10)

	raftPeers := []string{testRaftAddr, newRaftNode}
	nowRaftPeers, err := node.store.Peers()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nowRaftPeers, raftPeers) {
		t.Fatalf("leader node can not handle POST /join, now raft peers: %v\n", nowRaftPeers)
	}

	httpPeers := []string{testHTTPAddr, newNodeHTTP}
	nowHTTPPeers, err := master.Default.Peers()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nowHTTPPeers, httpPeers) {
		t.Fatalf("leader node can not handle POST /join, now http peers: %v\n", nowRaftPeers)
	}
}
