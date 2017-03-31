package client

import (
	"reflect"
	"testing"
	"time"
)

func mockCluster(addrs []string, err error) cluster {
	return clusterFunc(func() ([]string, error) {
		return addrs, err
	})
}

var caches = []string{"http://cache1.loc", "http://cache2.loc"}
var dbs = []string{"http://db1.loc", "http://db2.loc"}

func setDefaultMockCacheAndDB() {
	mockCacheAndDB(caches, dbs)
}

func mockCacheAndDB(caches []string, dbs []string) {
	cacheCluster = mockCluster(caches, nil)
	dbCluster = mockCluster(dbs, nil)
}

func TestNew(t *testing.T) {
	setDefaultMockCacheAndDB()

	cli, err := newClient()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(cli.caches, caches) {
		t.Errorf("failed to set caches, expect %v, got %v", caches, cli.caches)
	}

	if !reflect.DeepEqual(cli.dbs, dbs) {
		t.Errorf("failed to set dbs, expect %v, got %v", dbs, cli.dbs)
	}

	// test refresh
	newCaches := []string{"cache2.loc", "cache3.loc"}
	newDBs := []string{"db2.loc", "db3.loc"}
	mockCacheAndDB(newCaches, newDBs)

	// wait
	select {
	case <-time.After(time.Second * 2):
	}

	if !reflect.DeepEqual(cli.caches, newCaches) {
		t.Errorf("failed to set caches, expect %v, got %v", caches, cli.caches)
	}

	if !reflect.DeepEqual(cli.dbs, newDBs) {
		t.Errorf("failed to set dbs, expect %v, got %v", dbs, cli.dbs)
	}
}
