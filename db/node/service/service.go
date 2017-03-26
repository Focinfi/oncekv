// Package service provides the HTTP server for accessing the distributed key-value store.
// It also provides the endpoint for other nodes to join an existing cluster.
package service

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"encoding/json"

	"bytes"

	"github.com/Focinfi/oncekv/db/master"
	"github.com/Focinfi/oncekv/db/node/store"
	"github.com/Focinfi/oncekv/log"
	"github.com/Focinfi/oncekv/raftboltdb"
	"github.com/Focinfi/oncekv/utils/urlutil"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	logPrefix     = "db/node/service:"
	joinURLFormat = "%s/join"
)

type joinParams struct {
	Addr string `json:"addr"`
}

// Store is the interface Raft-backed key-value stores must implement.
type Store interface {
	// Open opens a store in a single mode or not
	Open(singleMode bool) error

	// Get returns the value for the given key.
	Get(key string) (string, error)

	// Add adds key/value, via distributed consensus.
	Add(key, value string) error

	// Join joins the node, reachable at addr, to the cluster.
	Join(addr string) error

	// Peers returns the store peers
	Peers() ([]string, error)

	// Leader returns the leader address
	Leader() string

	// Stats return the stats as a map[string]string
	Stats() map[string]string
}

// Service provides HTTP service.
type Service struct {
	// raft server
	raftAddr string
	ln       net.Listener

	//http server
	*gin.Engine
	httpAddr string

	// underlying store
	store Store
}

// New returns an uninitialized HTTP service.
func New(httpAddr string, raftAddr string, storeDir string) *Service {
	storage := store.New()
	storage.RaftBind = raftAddr
	storage.RaftDir = storeDir

	s := &Service{
		httpAddr: httpAddr,
		raftAddr: raftAddr,
		store:    storage,
		Engine:   gin.Default(),
	}

	s.GET("/i/key/:key", s.handleGet)
	s.POST("/key", s.handleSet)
	s.POST("/join", s.handleJoin)
	s.GET("/stats", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, s.store.Stats())
	})

	var wsupgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	s.GET("/ws/stats", func(ctx *gin.Context) {
		conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		defer conn.Close()

		for {
			select {
			case <-time.After(time.Second):
				b, err := json.Marshal(s.store.Stats())
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

	return s
}

// Start starts the service.
func (s *Service) Start() {
	if err := s.register(); err != nil {
		log.DB.Fatal(err)
	}

	peers, err := master.Default.Peers()
	if err != nil {
		log.DB.Fatal(err)
	}

	log.DB.Infoln(logPrefix, "Peers:", peers)

	if len(peers) == 0 {
		if err := s.store.Open(true); err != nil {
			log.DB.Fatal(err)
		}

		if err := master.Default.UpdatePeers([]string{s.httpAddr}); err != nil {
			log.DB.Fatal(err)
		}
	} else {
		if err := s.store.Open(false); err != nil {
			log.DB.Fatal(err)
		}
		if err := s.tryToJoin(peers); err != nil {
			log.DB.Fatal(err)
		}
	}

	log.DB.Fatal(s.Run(s.httpAddr))
}

func (s *Service) handleGet(ctx *gin.Context) {
	if s.raftAddr != s.store.Leader() {
		ctx.JSON(http.StatusBadRequest, StatusNotLeaderError)
		return
	}

	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, StatusParamsError)
		return
	}

	val, err := s.store.Get(key)
	if err != nil {
		fmt.Println(logPrefix, "Get Error: ", val, err)
		ctx.JSON(http.StatusInternalServerError, StatusInternalError)
		return
	}

	if val == "" {
		ctx.JSON(http.StatusNotFound, StatusKeyNotFound)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"key": key, "value": val})
}

func (s *Service) handleSet(ctx *gin.Context) {
	if s.raftAddr != s.store.Leader() {
		ctx.JSON(http.StatusBadRequest, StatusNotLeaderError)
		return
	}

	params := &struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}

	if err := ctx.BindJSON(params); err != nil {
		ctx.JSON(http.StatusOK, StatusParamsError)
		return
	}

	err := s.store.Add(params.Key, params.Value)
	if err == raftboltdb.ErrKeyDuplicated {
		ctx.JSON(http.StatusOK, StatusKeyDuplicate)
		return
	}

	if err != nil {
		log.DB.Error(err)
		ctx.JSON(http.StatusOK, StatusInternalError)
		return
	}

	ctx.JSON(http.StatusOK, StatusOK)
}

func (s *Service) handleJoin(ctx *gin.Context) {
	if s.raftAddr != s.store.Leader() {
		ctx.JSON(http.StatusBadRequest, StatusNotLeaderError)
		return
	}

	var remoteAddr = &joinParams{}

	if err := ctx.BindJSON(remoteAddr); err != nil {
		ctx.JSON(http.StatusBadRequest, StatusParamsError)
		return
	}

	if err := s.store.Join(remoteAddr.Addr); err != nil {
		ctx.JSON(http.StatusInternalServerError, StatusInternalError)
		return
	}

	ctx.JSON(http.StatusOK, StatusOK)

	go func() {
		if err := s.updatePeers(); err != nil {
			log.DB.Error(err)
		}
	}()
}

func (s *Service) tryToJoin(peers []string) error {
	if len(peers) == 0 {
		return nil
	}

	for _, peer := range peers {
		if peer == s.httpAddr {
			continue
		}

		params := &joinParams{Addr: s.raftAddr}
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}

		url := fmt.Sprintf(joinURLFormat, urlutil.MakeURL(peer))
		res, err := http.Post(url, "application/json", bytes.NewReader(b))
		if err != nil {
			log.DB.Error(err)
			continue
		}

		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("%s failed to join\n", logPrefix)
}

func (s *Service) updatePeers() error {
	raftPeers, err := s.store.Peers()
	if err != nil {
		return err
	}

	if len(raftPeers) == 0 {
		return master.Default.UpdatePeers([]string{s.httpAddr})
	}

	peers := []string{}
	for _, raftAddr := range raftPeers {
		peer, err := master.Default.PeerHTTPAddr(raftAddr)
		if err != nil {
			return err
		}

		peers = append(peers, peer)
	}

	return master.Default.UpdatePeers(peers)
}

func (s *Service) register() error {
	return master.Default.RegisterPeer(s.raftAddr, s.httpAddr)
}
