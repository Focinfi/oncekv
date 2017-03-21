package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"net/url"

	"github.com/Focinfi/oncekv/config"
)

func hostOfURL(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(fmt.Errorf("wrong url format, err: %v", err))
	}

	return u.Host
}

func httpGetterCluster(getterMap map[string]httpGetter) httpGetter {
	return httpGetterFunc(func(rawurl string) (*http.Response, error) {
		host := hostOfURL(rawurl)
		getter, ok := getterMap[host]
		if !ok {
			panic(fmt.Sprintf("client: no getter can handle %s", host))
		}

		return getter.Get(host)
	})
}

func mockHTTPGetter(url string, response string, err error, delay time.Duration) httpGetter {
	return httpGetterFunc(func(url string) (*http.Response, error) {
		respChan := make(chan *http.Response)

		go func() {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(response)),
			}

			time.AfterFunc(delay, func() {
				respChan <- resp
			})
		}()

		return <-respChan, err
	})
}

func httpPosterCluster(posterMap map[string]httpPoster) httpPoster {
	return httpPosterFunc(func(rawurl string, contentType string, body io.Reader) (*http.Response, error) {
		host := hostOfURL(rawurl)
		poster, ok := posterMap[host]
		if !ok {
			panic(fmt.Sprintf("client: no poster can handle %s", host))
		}

		return poster.Post(host, contentType, body)
	})
}

func mockHTTPPoster(url string, response string, err error, delay time.Duration) httpPoster {
	return httpPosterFunc(func(url string, contentType string, body io.Reader) (*http.Response, error) {
		respChan := make(chan *http.Response)

		go func() {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(response)),
			}

			time.AfterFunc(delay, func() {
				respChan <- resp
			})
		}()

		return <-respChan, err
	})
}

func defaultGetterCluster() httpGetter {
	cluster := map[string]httpGetter{}

	var servers []string
	servers = append(servers, caches...)
	servers = append(servers, dbs...)

	for i, cache := range servers {
		delay := config.Config().IdealKVResponseDuration * time.Duration(i%2)
		fmt.Printf("%s delay=%v\n", cache, delay)
		getter := mockHTTPGetter(cache, `{"key":"foo", "value":"bar"}`, nil, delay)
		host := hostOfURL(cache)
		cluster[host] = getter
	}

	return httpGetterCluster(cluster)
}

func defaultPosterCluster() httpPoster {
	cluster := map[string]httpPoster{}
	for i, db := range dbs {
		delay := config.Config().IdealKVResponseDuration * time.Duration(i%2)
		fmt.Printf("%s delay=%v\n", db, delay)
		poster := mockHTTPPoster(db, "", nil, delay)
		host := hostOfURL(db)
		cluster[host] = poster
	}

	return httpPosterCluster(cluster)
}

func setDefaultMockHTTP() {
	defaultGetter = defaultGetterCluster()
	defaultPoster = defaultPosterCluster()
}

func TestNewKV(t *testing.T) {
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, err := NewKV()
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
	setDefaultMockCacheAndDB()
	setDefaultMockHTTP()

	kv, _ := NewKV()
	_, err := kv.cache("foo")
	if err != nil {
		t.Fatal("can not fetch data from cache, err:", err)
	}

	// wait 1s
	select {
	case <-time.After(time.Millisecond * 100):
	}

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

	kv, _ := NewKV()
	_, err := kv.get("foo")
	if err != nil {
		t.Fatal(err)
	}

	//wait
	select {
	case <-time.After(time.Millisecond):
	}

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

	kv, _ := NewKV()
	err := kv.Put("foo1", "bar1")
	if err != nil {
		t.Fatal(err)
	}

	// wait
	select {
	case <-time.After(time.Millisecond):
	}

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
