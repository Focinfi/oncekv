package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"

	"time"

	groupcachehttp "github.com/Focinfi/oncekv/groupcache/http"
	"github.com/gin-gonic/gin"
	"github.com/golang/groupcache"
)

const basePath = "/sqs/"
const defaultGroup = "message"

type getter interface {
	Get(key string) string
}

// Node for one groupcahe server
type Node struct {
	sync.RWMutex
	addr   string
	master string
	dbs    []string
	peers  []string
	pool   *groupcachehttp.HTTPPool
	*gin.Engine
}

type masterParam struct {
	Peers []string `json:"peers"`
	DBs   []string `json:"dbs"`
}

// New returns a new Node with the addr
func New(addr string, masterAddr string) *Node {
	cache := &Node{
		master: strings.TrimSuffix(masterAddr, "/"),
		addr:   addr,
	}

	cache.Engine = newServer(cache)
	cache.pool = newPool(cache, addr)

	return cache
}

// Start starts the server
func (n *Node) Start() {
	n.join()
	n.Run(n.addr)
}

func newServer(c *Node) *gin.Engine {
	server := gin.Default()
	server.POST("/meta", c.handleMeta)
	server.GET("/sqs/message/:key", func(ctx *gin.Context) {
		c.pool.ServeHTTP(ctx.Writer, ctx.Request)
	})
	return server
}

func (n *Node) join() {
	// build join param
	b, err := json.Marshal(map[string]string{"addr": n.addr})
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
			if err != nil {
				log.Println("fetchData err: ", err)
			}
			defer res.Body.Close()

			val, err := ioutil.ReadAll(res.Body)

			n.Lock()
			defer n.Unlock()

			completedCount++
			if completedCount == len(dbs) {
				if res.StatusCode == http.StatusOK {
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

		return errors.New("not found")
	}
}

func newPool(node *Node, addr string) *groupcachehttp.HTTPPool {
	pool := groupcachehttp.NewHTTPPoolOpts("http://"+addr,
		&groupcachehttp.HTTPPoolOptions{
			BasePath: basePath,
		})

	newGroup(node, "message")
	return pool
}

func newGroup(n *Node, name string) {
	// TODO: make cacheBizes to be configurable
	groupcache.NewGroup(name, 1<<32, groupcache.GetterFunc(n.fetchData))
}
