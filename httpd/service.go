// Package httpd provides the HTTP server for accessing the distributed key-value store.
// It also provides the endpoint for other nodes to join an existing cluster.
package httpd

import (
	"fmt"
	"log"
	"net"
	"net/http"

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
}

// Service provides HTTP service.
type Service struct {
	addr string
	ln   net.Listener
	*gin.Engine

	store Store
}

// New returns an uninitialized HTTP service.
func New(addr string, store Store) *Service {
	s := &Service{
		addr:   addr,
		store:  store,
		Engine: gin.Default(),
	}

	s.GET("/i/key/:key", s.handleGet)
	s.POST("/key", s.handleSet)
	s.POST("/join", s.handleJoin)

	return s
}

// Start starts the service.
func (s *Service) Start() {
	go func() {
		if err := s.Run(s.addr); err != nil {
			panic(err)
		}
	}()
}

func (s *Service) handleGet(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, StatusParamsError)
		return
	}

	val, err := s.store.Get(key)
	if err != nil {
		ctx.JSON(http.StatusOK, StatusInternalError)
		return
	}

	if val == "" {
		ctx.JSON(http.StatusOK, StatusKeyNotFound)
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
}
