package oncekv

import (
	"os"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/cache/node"
	"github.com/Focinfi/oncekv/client"
	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/db/node/service"
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

	time.Sleep(time.Second)
	// create a single raft cluster
	go service.New(":55463", ":55464", testDataDir).Start()

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
