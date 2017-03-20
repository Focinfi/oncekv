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

var caches = []string{"cache1.loc", "cache2.loc"}
var dbs = []string{"db1.loc", "db2.loc"}

func TestNew(t *testing.T) {
	cacheCluster = mockCluster(caches, nil)
	dbCluster = mockCluster(dbs, nil)

	client, err := New()
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
	done := make(chan bool)
	go func() {
		caches = []string{"cache2.loc", "cache3.loc"}
		dbs = []string{"db2.loc", "db3.loc"}
		cacheCluster = mockCluster(caches, nil)
		dbCluster = mockCluster(dbs, nil)
		time.AfterFunc(config.Config().OncekvMetaRefreshPeroid*2, func() {
			done <- true
		})
	}()
	<-done

	if !reflect.DeepEqual(client.caches, caches) {
		t.Errorf("failed to set caches, expect %v, got %v", caches, client.caches)
	}

	if !reflect.DeepEqual(client.dbs, dbs) {
		t.Errorf("failed to set dbs, expect %v, got %v", dbs, client.dbs)
	}
}
