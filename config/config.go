package config

import (
	"errors"
	"io"
	"os"

	"time"

	"github.com/gin-gonic/gin"
)

const (
	testEnv       = "test"
	developEnv    = "develop"
	productionEnv = "production"
)

const (
	test       = "test"
	develop    = "develop"
	production = "production"
)

var (
	// ErrDataNotFound error for data not found
	ErrDataNotFound = errors.New("oncekv: data not found")
)

// Envroinment for application envroinment
type Envroinment string

// IsProduction returns if the env equals to production
func (e Envroinment) IsProduction() bool {
	return e == production
}

// IsDevelop returns if the env equals to develop
func (e Envroinment) IsDevelop() bool {
	return e == develop
}

// IsTest returns if the env equals to develop
func (e Envroinment) IsTest() bool {
	return e == test
}

var env = develop

// Env returns the env
func Env() Envroinment {
	return Envroinment(env)
}

// Configuration defines configuration
type Configuration struct {
	LogOut                  io.Writer
	EtcdEndpoints           []string
	OncekvMetaRefreshPeroid time.Duration
	HTTPRequestTimeout      time.Duration
	IdealResponseDuration   time.Duration
	CacheBytes              int64
	CacheMasterAddr         string
	// RaftNodesKey for raft ndoes store key
	RaftNodesKey string
	// CacheNodesKey for cache nodes store key
	CacheNodesKey string

	RaftKey string
}

func newDefaultConfig() Configuration {
	return Configuration{
		LogOut:                  os.Stdout,
		EtcdEndpoints:           []string{"localhost:2379"},
		OncekvMetaRefreshPeroid: time.Second,
		HTTPRequestTimeout:      time.Millisecond * 100,
		IdealResponseDuration:   time.Millisecond * 50,
		CacheBytes:              1 << 20,
		CacheMasterAddr:         ":5550",
		RaftNodesKey:            "oncekv.db.nodes",
		CacheNodesKey:           "oncekv.cache.nodes",
		RaftKey:                 "oncekv.nodes.http.adrr",
	}
}

// Config returns the Configuration based on envroinment
func Config() Configuration {

	switch env {
	case productionEnv:
		return Configuration{
			LogOut:                  os.Stdout,
			OncekvMetaRefreshPeroid: time.Second,
			HTTPRequestTimeout:      time.Millisecond * 100,
			IdealResponseDuration:   time.Millisecond * 50,
			CacheBytes:              1 << 32,
			CacheMasterAddr:         ":5550",
		}
	case developEnv:
		return newDefaultConfig()
	default:
		return newDefaultConfig()
	}
}

func init() {
	if e := os.Getenv("ONCEKV_ENV"); e != "" {
		env = e
	}

	if Env().IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
}
