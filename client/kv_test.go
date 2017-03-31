package client

import (
	"fmt"
	"testing"
	"time"

	"github.com/Focinfi/oncekv/utils/mock"
)

func defaultGetterCluster() mock.HTTPGetter {
	cluster := map[string]mock.HTTPGetter{}

	var servers []string
	servers = append(servers, caches...)
	servers = append(servers, dbs...)

	for i, cache := range servers {
		delay := idealResponseDuration * time.Duration(i%2)
		fmt.Printf("%s delay=%v\n", cache, delay)
		getter := mock.MakeHTTPGetter(cache, `{"key":"foo", "value":"bar"}`, nil, delay)
		host := mock.HostOfURL(cache)
		cluster[host] = getter
	}

	return mock.HTTPGetterCluster(cluster)
}

func defaultPosterCluster() mock.HTTPPoster {
	cluster := map[string]mock.HTTPPoster{}
	for i, db := range dbs {
		delay := idealResponseDuration * time.Duration(i%2)
		fmt.Printf("%s delay=%v\n", db, delay)
		poster := mock.MakeHTTPPoster(db, "", nil, delay)
		host := mock.HostOfURL(db)
		cluster[host] = poster
	}

	return mock.HTTPPosterCluster(cluster)
}

func setDefaultMockHTTP() {
	defaultGetter = defaultGetterCluster()
	defaultPoster = defaultPosterCluster()
}

func TestNewKV(t *testing.T) {
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, err := DefaultKV()
	if err != nil {
		t.Fatal(err)
	}

	if kv.cli.fastCache != "" {
		t.Errorf("set fastCache at the beginning: %s", kv.cli.fastCache)
	}

	if kv.cli.fastDB != "" {
		t.Errorf("set fastCache at the beginning: %s", kv.cli.fastDB)
	}
}

func TestCache(t *testing.T) {
	t.Log(requestTimeout, idealResponseDuration)
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, _ := DefaultKV()
	_, err := kv.cache("foo")
	if err != nil {
		t.Fatal("can not fetch data from cache, err:", err)
	}

	time.Sleep(requestTimeout * 2)

	if kv.cli.fastCache != caches[0] {
		t.Errorf("can not set the right fastCache, expect: %s, go: %v", caches[0], kv.cli.fastCache)
	}

	// test fastCache
	respErr := make(chan error)
	go func() {
		_, err = kv.cache("foo")
		respErr <- err
	}()

	select {
	case <-time.After(time.Millisecond):
		t.Error("can not use cache")
	case err := <-respErr:
		if err != nil {
			t.Fatal("can not fetch data from cache, err:", err)
		}
	}
}

func TestGet(t *testing.T) {
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, _ := DefaultKV()
	_, err := kv.get("foo")
	if err != nil {
		t.Fatal(err)
	}

	//wait
	time.Sleep(requestTimeout)

	if kv.cli.fastDB != dbs[0] {
		t.Errorf("can not set fastDB, expect %s, got %s\n", dbs[0], kv.cli.fastDB)
	}

	// test fastDB
	respErr := make(chan error)
	go func() {
		_, err = kv.get("foo")
		respErr <- err
	}()

	select {
	case <-time.After(time.Millisecond):
		t.Error("can not use fastDB")
	case err := <-respErr:
		if err != nil {
			t.Fatal("can not fetch data from db, err:", err)
		}
	}
}

func TestPut(t *testing.T) {
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, _ := DefaultKV()
	err := kv.Put("foo1", "bar1")
	if err != nil {
		t.Fatal(err)
	}

	// wait
	time.Sleep(time.Millisecond)

	if kv.cli.fastDB != dbs[0] {
		t.Errorf("can not set fastDB, expect %s, got %s\n", dbs[0], kv.cli.fastDB)
	}

	// test fastDB
	respErr := make(chan error)
	go func() {
		err = kv.Put("foo2", "bar2")
		respErr <- err
	}()

	select {
	case <-time.After(time.Millisecond):
		t.Error("can not use fastDB")
	case err := <-respErr:
		if err != nil {
			t.Fatal("can not set kv into db, err:", err)
		}
	}
}
