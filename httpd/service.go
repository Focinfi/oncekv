// Package httpd provides the HTTP server for accessing the distributed key-value store.
// It also provides the endpoint for other nodes to join an existing cluster.
package httpd

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/Focinfi/oncekv/master"
	"github.com/Focinfi/oncekv/raftboltdb"
	"github.com/gin-gonic/gin"
)

// Store is the interface Raft-backed key-value stores must implement.
type Store interface {
	// Get returns the value for the given key.
	Get(key string) (string, error)

	// Set sets the value for the given key, via distributed consensus.
	Set(key, value string) error

	// Delete removes the given key, via distributed consensus.
	Delete(key string) error

	// Join joins the node, reachable at addr, to the cluster.
	Join(addr string) error

	// Peers returns the store peers
	Peers() ([]string, error)
}

// Service provides HTTP service.
type Service struct {
	httpAddr string
	raftAddr string
	ln       net.Listener
	*gin.Engine

	store Store
}

// New returns an uninitialized HTTP service.
func New(httpAddr string, raftAddr string, store Store) *Service {
	s := &Service{
		httpAddr: httpAddr,
		raftAddr: raftAddr,
		store:    store,
		Engine:   gin.Default(),
	}

	s.GET("/i/key/:key", s.handleGet)
	s.POST("/key", s.handleSet)
	s.POST("/join", s.handleJoin)

	return s
}

// Start starts the service.
func (s *Service) Start() {
	log.Println("Try to start")
	go func() {
		if err := s.Run(s.httpAddr); err != nil {
			panic(err)
		}
	}()

	if err := s.register(); err != nil {
		panic(err)
	}

	log.Println("Try to update peers")
	if err := s.updatePeers(); err != nil {
		log.Println("oncekv httpd: failed to update peers")
	}
}

func (s *Service) handleGet(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, StatusParamsError)
		return
	}

	val, err := s.store.Get(key)
	if err != nil {
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
	fmt.Println("GET key")
	params := &struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}

	if err := ctx.BindJSON(params); err != nil {
		ctx.JSON(http.StatusOK, StatusParamsError)
		return
	}

	err := s.store.Set(params.Key, params.Value)
	if err == raftboltdb.ErrKeyDuplicated {
		ctx.JSON(http.StatusOK, StatusKeyDuplicate)
		return
	}

	if err != nil {
		log.Println(err)
		ctx.JSON(http.StatusOK, StatusInternalError)
		return
	}

	ctx.JSON(http.StatusOK, StatusOK)
}

func (s *Service) handleJoin(ctx *gin.Context) {
	var remoteAddr = &struct {
		Addr string `json:"addr"`
	}{}

	if err := ctx.BindJSON(remoteAddr); err != nil {
		ctx.JSON(http.StatusOK, StatusParamsError)
		return
	}

	if err := s.store.Join(remoteAddr.Addr); err != nil {
		ctx.JSON(http.StatusOK, StatusInternalError)
		return
	}

	ctx.JSON(http.StatusOK, StatusOK)

	go func() {
		if err := s.updatePeers(); err != nil {
			log.Println(err)
		}
	}()
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
