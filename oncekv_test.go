package oncekv

import (
	"testing"

	"os"

	"github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/cache/node"
	"github.com/Focinfi/oncekv/client"
	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/db/node/httpd"
	"github.com/Focinfi/oncekv/db/node/store"
)

const testDataDir = "test_data"

func BasicTest(t *testing.T) {
	os.Mkdir(testDataDir, os.ModeAppend)
	// clean test data
	defer func() {
		os.RemoveAll(testDataDir)
	}()

	// start cache master
	go master.Default.Start()
	// start cache node
	go node.New(":55461", ":54461", config.Config().CacheMasterAddr)

	// create a single raft cluster
	s := store.New()
	s.RaftDir = testDataDir
	s.RaftBind = ":55441"
	go httpd.New(s.RaftBind, ":55441", s)

	// create client
	cli, err := client.DefaultKV()
	if err != nil {
		t.Fatal(err)
	}

	// set foo/bar
	cli.Put("foo", "bar")
	// get foo
	val, err := cli.Get("foo")
	if err != nil {
		t.Fatal(err)
	}

	if val != "bar" {
		t.Errorf("oncekv: get expect bar, got %v\n", val)
	}
}
