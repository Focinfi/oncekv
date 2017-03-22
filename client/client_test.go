package client

import (
	"reflect"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/config"
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

	client, err := newClient()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(client.caches, caches) {
		t.Errorf("failed to set caches, expect %v, got %v", caches, client.caches)
	}

	if !reflect.DeepEqual(client.dbs, dbs) {
		t.Errorf("failed to set dbs, expect %v, got %v", dbs, client.dbs)
	}

	// test refresh
	newCaches := []string{"cache2.loc", "cache3.loc"}
	newDBs := []string{"db2.loc", "db3.loc"}
	mockCacheAndDB(newCaches, newDBs)

	// wait
	select {
	case <-time.After(config.Config().OncekvMetaRefreshPeroid * 2):
	}

	if !reflect.DeepEqual(client.caches, newCaches) {
		t.Errorf("failed to set caches, expect %v, got %v", caches, client.caches)
	}

	if !reflect.DeepEqual(client.dbs, newDBs) {
		t.Errorf("failed to set dbs, expect %v, got %v", dbs, client.dbs)
	}
}
