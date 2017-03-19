package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/sqs/util/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/golang/groupcache"
)

const (
	jsonHTTPHeader      = "application-type/json"
	basePath            = "/sqs/"
	defaultGroup        = "message"
	masterJoinURLFormat = "%s/join"
	dbGetURLFormat      = "%s/i/key/%s"
	dbQueryTimeout      = time.Millisecond * 300
	gorupCacheBytes     = 1 << 32
)

var (
	// ErrDataNotFound for not found data error
	ErrDataNotFound = errors.New("groupcache node: data not found")
	// ErrDatabaseQueryTimeout for upderlying data query timeout error
	ErrDatabaseQueryTimeout = errors.New("gropcache node: upderlying data query timeout")
)

type masterParam struct {
	Peers []string `json:"peers"`
	DBs   []string `json:"dbs"`
}

// Node for one groupcahe server
type Node struct {
	// meta
	sync.RWMutex // protect updating for dbs and peers
	// upderlying databse
	dbs []string
	// cache peers
	peers []string
	// master url for update meta(dbs and peers)
	masterAdrr string

	// node http server
	*gin.Engine
	httpAddr string

	// groupcache server
	nodeAddr string
	pool     *groupcache.HTTPPool
	group    *groupcache.Group
}

// New returns a new Node with the given info
func New(httpAddr string, nodeAddr string, masterAddr string) *Node {
	cache := &Node{
		masterAdrr: strings.TrimSuffix(masterAddr, "/"),
		httpAddr:   httpAddr,
		nodeAddr:   nodeAddr,
	}

	cache.Engine = newServer(cache)
	cache.pool = newPool(cache, nodeAddr)
	cache.group = newGroup(cache, defaultGroup)

	return cache
}

// Start starts the server
func (n *Node) Start() {
	// try to get meta data
	n.join()

	// start the groupcache server
	go func() {
		log.DB.Fatal(http.ListenAndServe(n.nodeAddr, n.pool))
	}()

	// start the node server
	n.Run(n.httpAddr)
}

func newServer(c *Node) *gin.Engine {
	server := gin.Default()
	server.POST("/meta", c.handleMeta)
	server.GET("/key/:key", func(ctx *gin.Context) {
		result := &groupcache.ByteView{}
		err := c.group.Get(ctx.Request.Context(), ctx.Param("key"), groupcache.ByteViewSink(result))
		if err == ErrDataNotFound {
			ctx.JSON(http.StatusNotFound, nil)
			return
		}

		if err != nil {
			log.DB.Error(err)
			ctx.JSON(http.StatusInternalServerError, nil)
			return
		}

		ctx.Writer.Write(result.ByteSlice())
		ctx.Writer.WriteHeader(http.StatusOK)
	})
	return server
}

func newPool(node *Node, addr string) *groupcache.HTTPPool {
	return groupcache.NewHTTPPoolOpts(urlutil.MakeURL(addr),
		&groupcache.HTTPPoolOptions{
			BasePath: basePath,
		})
}

func newGroup(n *Node, name string) *groupcache.Group {
	// TODO: make cacheBizes to be configurable
	return groupcache.NewGroup(name, gorupCacheBytes, groupcache.GetterFunc(n.fetchData))
}

func (n *Node) join() {
	// build join param
	b, err := json.Marshal(map[string]string{
		"httpAddr": n.httpAddr,
		"nodeAddr": n.nodeAddr,
	})
	if err != nil {
		panic(err)
	}

	// post join
	res, err := http.Post(fmt.Sprintf(masterJoinURLFormat, n.masterAdrr), jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		panic("groupcache node: failed to join into master")
	}

	defer res.Body.Close()

	// read reponse
	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	params := masterParam{}
	if err := json.Unmarshal(b, &params); err != nil {
		panic(err)
	}

	n.Lock()
	defer n.Unlock()

	// update meta
	n.pool.Set(params.Peers...)
	n.peers = params.Peers
	n.dbs = params.DBs
}

func (n *Node) handleMeta(ctx *gin.Context) {
	params := masterParam{}
	if err := ctx.BindJSON(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sort.StringSlice(params.Peers).Sort()
	sort.StringSlice(params.DBs).Sort()

	n.RLock()
	if reflect.DeepEqual(n.peers, params.Peers) &&
		reflect.DeepEqual(n.dbs, params.DBs) {

		n.RUnlock()
		// return if no changes
		ctx.JSON(http.StatusOK, nil)
		return
	}
	n.RUnlock()

	log.Biz.Infof("%#v, %#v\n", n.peers, params.Peers)
	log.Biz.Infof("%#v, %#v\n", n.dbs, params.DBs)

	n.Lock()
	defer n.Unlock()

	n.pool.Set(params.Peers...)
	n.peers = params.Peers
	n.dbs = params.DBs

	ctx.JSON(http.StatusOK, nil)
}

func (n *Node) fetchData(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	var dbs = make([]string, len(n.dbs))
	copy(dbs, n.dbs)
	if len(dbs) == 0 {
		log.DB.Errorln("groupcache node: no database")
		return ErrDatabaseQueryTimeout
	}

	var data = make(chan []byte)
	var done bool
	var completedCount int

	for _, db := range dbs {
		url := urlutil.MakeURL(db)
		go func() {
			url = fmt.Sprintf(dbGetURLFormat, url, key)
			resp, err := http.Get(url)
			var val []byte
			var statusCode int

			if err != nil {
				log.DB.Errorf("fetchData err: %s\n", err)
			} else {
				// read the res.Body only if err == nil
				defer resp.Body.Close()

				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.DB.Info(err)
				}

				statusCode = resp.StatusCode
				if statusCode == http.StatusOK {
					val = data
				}
			}

			n.Lock()
			defer n.Unlock()

			completedCount++
			if statusCode == http.StatusOK || completedCount == len(dbs) {
				if !done {
					go func() { data <- val }()
					done = true
				}
			}
		}()
	}

	select {
	case <-time.After(dbQueryTimeout):
		return ErrDatabaseQueryTimeout
	case val := <-data:
		log.Biz.Debugf("Val: '%s'\n", string(val))
		if len(val) > 0 {
			dest.SetString(string(val))
			return nil
		}

		return ErrDataNotFound
	}
}
