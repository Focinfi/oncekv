package oncekv

import (
	"os"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/cache/node"
	"github.com/Focinfi/oncekv/client"
	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/db/node/httpd"
	"github.com/Focinfi/oncekv/db/node/store"
)

const testDataDir = "test_data"

func TestBasic(t *testing.T) {
	os.RemoveAll(testDataDir)
	os.Mkdir(testDataDir, 0711)
	// clean test data
	defer func() {
		os.RemoveAll(testDataDir)
	}()

	// start cache master
	go master.Default.Start()
	// start cache node
	go node.New(":55461", ":55462", config.Config().CacheMasterAddr).Start()

	// create a single raft cluster
	s := store.New()
	s.RaftDir = testDataDir
	s.RaftBind = ":55464"
	if err := s.Open(true); err != nil {
		t.Fatal(err)
	}
	go httpd.New(":55463", s.RaftBind, s).Start()

	time.Sleep(time.Second * 2)

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

	time.Sleep(time.Second)

	if val != "bar" {
		t.Errorf("oncekv: get expect bar, got %v\n", val)
	}
}
