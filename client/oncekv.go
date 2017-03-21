package client

import (
	"reflect"
	"sort"
	"sync"
	"time"

	cachemaster "github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/log"
	oncekvmaster "github.com/Focinfi/oncekv/master"
)

type cluster interface {
	Peers() ([]string, error)
}

type clusterFunc func() ([]string, error)

func (c clusterFunc) Peers() ([]string, error) {
	return c()
}

var dbCluster = cluster(oncekvmaster.Default)

var cacheCluster = cluster(clusterFunc(cachemaster.Peers))

// Client for requesting oncekv
type Client struct {
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

// New returns a new Client and ready to use
func New() (*Client, error) {
	client := &Client{
		dbs:    []string{},
		caches: []string{},
	}

	if err := client.update(); err != nil {
		return nil, err
	}

	go client.refresh()

	return client, nil
}

func (c *Client) setFastCache(cacheURL string) {
	c.Lock()
	defer c.Unlock()

	c.fastCache = cacheURL
}

func (c *Client) setFastDB(dbURL string) {
	c.Lock()
	defer c.Unlock()

	c.fastDB = dbURL
}

func (c *Client) refresh() {
	ticker := time.NewTicker(config.Config().OncekvMetaRefreshPeroid)
	for {
		select {
		case <-ticker.C:
			if err := c.update(); err != nil {
				log.DB.Error(err)
			}
		}
	}
}

func (c *Client) update() error {
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
