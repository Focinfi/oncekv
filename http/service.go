// Package httpd provides the HTTP server for accessing the distributed key-value store.
// It also provides the endpoint for other nodes to join an existing cluster.
package httpd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Store is the interface Raft-backed key-value stores must implement.
type Store interface {
	// Get returns the value for the given key.
	Get(key string) (string, error)

	// Set sets the value for the given key, via distributed consensus.
	Set(key, value string) error

	// Delete removes the given key, via distributed consensus.
	// Delete(key string) error

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

	s.GET("/key/:key", func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, nil)
			return
		}

		val, err := s.store.Get(key)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, err)
			return
		}

		if val == "" {
			ctx.JSON(http.StatusBadRequest, nil)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"key": key, "value": val})
	})

	s.POST("/key", func(ctx *gin.Context) {
		fmt.Println("GET key")
		params := &struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}{}

		if err := ctx.BindJSON(params); err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		if err := s.store.Set(params.Key, params.Value); err != nil {
			ctx.JSON(http.StatusBadRequest, err)
		}

		ctx.JSON(http.StatusOK, nil)
	})

	return s
}

// Start starts the service.
func (s *Service) Start() error {
	return s.Run(s.addr)
}

// ServeHTTP allows Service to serve HTTP requests.
func (s *Service) handleJoin(w http.ResponseWriter, r *http.Request) {
	m := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(m) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr, ok := m["addr"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.store.Join(remoteAddr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
