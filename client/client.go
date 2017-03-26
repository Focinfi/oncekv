package client

import (
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/admin"
	"github.com/Focinfi/oncekv/log"
)

type cluster interface {
	Peers() ([]string, error)
}

type clusterFunc func() ([]string, error)

func (c clusterFunc) Peers() ([]string, error) {
	return c()
}

var (
	dbCluster    = cluster(admin.Default.DBMaster)
	cacheCluster = cluster(admin.Default.CacheMaster)
)

// lient for requesting oncekv
type client struct {
	// meta
	sync.RWMutex
	dbs    []string
	caches []string
	// last fast enough cache server URL
	fastCache string
	// last fast enough database server URL
	fastDB string

	// meta server
	cacheCluster cluster
	dbCluster    cluster
}

func newClient() (*client, error) {
	client := &client{
		dbs:    []string{},
		caches: []string{},
	}

	if err := client.update(); err != nil {
		return nil, err
	}

	go client.refresh()

	return client, nil
}

func (c *client) setFastCache(cacheURL string) {
	c.Lock()
	defer c.Unlock()

	c.fastCache = cacheURL
}

func (c *client) setFastDB(dbURL string) {
	c.Lock()
	defer c.Unlock()

	c.fastDB = dbURL
}

func (c *client) refresh() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if err := c.update(); err != nil {
				log.DB.Error(err)
			}
		}
	}
}

func (c *client) update() error {
	dbs, err := dbCluster.Peers()
	if err != nil {
		return err
	}
	sort.StringSlice(dbs).Sort()

	caches, err := cacheCluster.Peers()
	if err != nil {
		return err
	}
	sort.StringSlice(caches).Sort()

	log.Biz.Infoln(logPrefix, dbs, caches)

	c.RLock()
	if reflect.DeepEqual(c.dbs, dbs) &&
		reflect.DeepEqual(c.caches, caches) {
		c.RUnlock()
		return nil

	}
	c.RUnlock()

	c.Lock()
	defer c.Unlock()
	c.dbs = dbs
	c.caches = caches

	return nil
}
