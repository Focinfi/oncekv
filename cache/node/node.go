package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Focinfi/oncekv/config"
	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/utils/mock"
	"github.com/Focinfi/oncekv/utils/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/golang/groupcache"
	"github.com/gorilla/websocket"
)

const (
	jsonHTTPHeader      = "application-type/json"
	basePath            = "/oncekv/"
	defaultGroup        = "kv"
	masterJoinURLFormat = "%s/join"
	dbGetURLFormat      = "%s/i/key/%s"
	logPrefix           = "cache/node:"
)

var (
	// ErrDataNotFound for not found data error
	ErrDataNotFound = fmt.Errorf("%s data not found", logPrefix)
	// ErrDatabaseQueryTimeout for upderlying data query timeout error
	ErrDatabaseQueryTimeout = fmt.Errorf("%s upderlying data query timeout", logPrefix)

	dbQueryTimeout  = config.Config.HTTPRequestTimeout
	gorupCacheBytes = config.Config.CacheBytes

	httpGetter = mock.HTTPGetter(mock.HTTPGetterFunc(http.Get))
	httpPoster = mock.HTTPPoster(mock.HTTPPosterFunc(http.Post))
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
	// fast db
	fastDB string
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
		log.DB.Fatal(logPrefix, http.ListenAndServe(n.nodeAddr, n.pool))
	}()

	// start the node server
	n.Run(n.httpAddr)
}

func newServer(c *Node) *gin.Engine {
	server := gin.Default()
	server.POST("/meta", c.handleMeta)

	server.GET("/stats", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, c.group.Stats)
	})

	server.GET("/key/:key", func(ctx *gin.Context) {
		result := &groupcache.ByteView{}
		log.DB.Infoln("Start Get")
		err := c.group.Get(ctx.Request.Context(), ctx.Param("key"), groupcache.ByteViewSink(result))
		log.DB.Infoln("End Get")
		if err == ErrDataNotFound {
			ctx.JSON(http.StatusNotFound, nil)
			return
		}

		if err != nil {
			log.DB.Error(logPrefix, err)
			ctx.JSON(http.StatusInternalServerError, nil)
			return
		}

		ctx.Writer.WriteHeader(http.StatusOK)
		ctx.Writer.Header()["Content-Type"] = []string{"application/json; charset=utf-8"}
		ctx.Writer.Write(result.ByteSlice())
	})

	var wsupgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server.GET("/ws/stats", func(ctx *gin.Context) {
		conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		defer conn.Close()

		for {
			select {
			case <-time.After(time.Second):
				b, err := json.Marshal(c.group.Stats)
				if err != nil {
					log.DB.Error(err)
					continue
				}

				if err := conn.WriteMessage(1, b); err != nil {
					return
				}
			}
		}
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
	addr := urlutil.MakeURL(n.masterAdrr)
	response, err := httpPoster.Post(fmt.Sprintf(masterJoinURLFormat, addr), jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("%s failed to join into master", logPrefix))
	}

	// read reponse
	b, err = ioutil.ReadAll(response.Body)
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

	log.Biz.Infof("%s [peers] local:%#v, remote: %#v\n", logPrefix, n.peers, params.Peers)
	log.Biz.Infof("%s [dbs] local:%#v, remote: %#v\n", logPrefix, n.dbs, params.DBs)

	n.Lock()
	defer n.Unlock()

	n.pool.Set(params.Peers...)
	n.peers = params.Peers
	n.dbs = params.DBs

	ctx.JSON(http.StatusOK, nil)
}

func (n *Node) fetchData(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	if n.fastDB == "" {
		return n.tryAllDBfind(ctx, key, dest)
	}

	data, err := n.find(key, n.fastDB)
	if err == ErrDataNotFound {
		return err
	}

	if err != nil {
		log.DB.Error(logPrefix, err)
		go n.setFastDB("")
		return n.tryAllDBfind(ctx, key, dest)
	}

	dest.SetBytes(data)
	return nil
}

func (n *Node) find(key string, url string) ([]byte, error) {
	url = fmt.Sprintf(dbGetURLFormat, urlutil.MakeURL(url), key)
	resp, err := httpGetter.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrDataNotFound
	}

	if resp.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if len(b) == 0 {
			return nil, fmt.Errorf("%s database error, lost data of key: %s\n", logPrefix, key)
		}

		return b, nil
	}

	return nil, fmt.Errorf("%s failed to fetch data", logPrefix)
}

func (n *Node) tryAllDBfind(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	dbs := make([]string, len(n.dbs))
	copy(dbs, n.dbs)
	log.Biz.Infoln(logPrefix, "start fetchData:", time.Now(), dbs)
	if len(dbs) == 0 {
		return fmt.Errorf("%s databases are not available\n", logPrefix)
	}

	var got bool
	var data = make(chan []byte)
	var completeCount int
	var fastURL string
	var resErr error

	for _, db := range dbs {
		go func(url string) {
			val, err := n.find(key, url)

			n.Lock()
			defer n.Unlock()
			if len(val) > 0 || err == ErrDataNotFound || completeCount == len(dbs) {
				if !got {
					got = true
					fastURL = url
					resErr = err

					go func() { data <- val }()
				}
			}
		}(db)
	}

	select {
	case <-time.After(dbQueryTimeout):
		go n.setFastDB("")
		return ErrDatabaseQueryTimeout

	case value := <-data:
		log.Biz.Infoln(logPrefix, "end get:", time.Now())
		dest.SetBytes(value)

		if len(value) > 0 || resErr == ErrDataNotFound {
			go n.setFastDB(fastURL)
		}

		return resErr
	}
}

func (n *Node) setFastDB(db string) {
	n.Lock()
	defer n.Unlock()

	n.fastDB = db
}
