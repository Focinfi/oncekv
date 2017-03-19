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

const basePath = "/sqs/"
const defaultGroup = "message"

var ErrDataNotFound = errors.New("groupcache: data not found")

type getter interface {
	Get(key string) string
}

// Node for one groupcahe server
type Node struct {
	sync.RWMutex
	httpAddr string
	nodeAddr string
	master   string
	dbs      []string
	peers    []string
	pool     *groupcache.HTTPPool
	group    *groupcache.Group
	*gin.Engine
}

type masterParam struct {
	Peers []string `json:"peers"`
	DBs   []string `json:"dbs"`
}

// New returns a new Node with the addr
func New(httpAddr string, nodeAddr string, masterAddr string) *Node {
	cache := &Node{
		master:   strings.TrimSuffix(masterAddr, "/"),
		httpAddr: httpAddr,
		nodeAddr: nodeAddr,
	}

	cache.Engine = newServer(cache)
	cache.pool = newPool(cache, nodeAddr)
	cache.group = newGroup(cache, "message")

	return cache
}

// Start starts the server
func (n *Node) Start() {
	n.join()
	go http.ListenAndServe(n.nodeAddr, n.pool)
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
	res, err := http.Post(fmt.Sprintf("%s/join", n.master), "application-type/json", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		panic("failed to join")
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

	// update peers
	n.pool.Set(params.Peers...)
	n.peers = params.Peers
	n.dbs = params.DBs
}

func (n *Node) handleMeta(ctx *gin.Context) {
	params := masterParam{}
	if err := ctx.BindJSON(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	sort.StringSlice(params.Peers).Sort()
	sort.StringSlice(params.DBs).Sort()

	n.RLock()
	if reflect.DeepEqual(n.peers, params.Peers) &&
		reflect.DeepEqual(n.dbs, params.DBs) {
		n.RUnlock()
		ctx.JSON(http.StatusOK, nil)
		return
	}
	n.RUnlock()

	fmt.Printf("%#v, %#v\n", n.peers, params.Peers)
	fmt.Printf("%#v, %#v\n", n.dbs, params.DBs)
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
		return errors.New("no database")
	}

	var data = make(chan []byte)
	var done bool
	var completedCount int

	for _, db := range dbs {
		url := competeAddr(db)
		go func() {
			url = fmt.Sprintf("%s/i/key/%s", strings.TrimSuffix(url, "/"), key)
			fmt.Println("URL: ", url)
			res, err := http.Get(url)
			var val []byte
			var statusCode int
			if err != nil {
				log.DB.Println("fetchData err: ", err)
			} else {
				val, err = ioutil.ReadAll(res.Body)
				statusCode = res.StatusCode
				res.Body.Close()
			}

			n.Lock()
			defer n.Unlock()

			completedCount++
			if completedCount == len(dbs) {
				if statusCode == http.StatusOK {
					go func() { data <- val }()
					return
				}

				go func() { data <- nil }()
				return
			}

			if res.StatusCode == http.StatusOK {
				if !done {
					done = true
					go func() { data <- val }()
				}
			}

		}()
	}

	select {
	case <-time.After(time.Millisecond * 300):
		fmt.Println("timeout")
		return errors.New("timeout")
	case val := <-data:
		fmt.Printf("Val: '%s'", string(val))
		if len(val) > 0 {
			dest.SetString(string(val))
			return nil
		}

		return ErrDataNotFound
	}
}

func newPool(node *Node, addr string) *groupcache.HTTPPool {
	pool := groupcache.NewHTTPPoolOpts(urlutil.MakeURL(addr),
		&groupcache.HTTPPoolOptions{
			BasePath: basePath,
		})

	return pool
}

func newGroup(n *Node, name string) *groupcache.Group {
	// TODO: make cacheBizes to be configurable
	return groupcache.NewGroup(name, 1<<32, groupcache.GetterFunc(n.fetchData))
}
