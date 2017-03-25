package config

import (
	"errors"
	"io"
	"os"

	"time"

	"path"

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
var root = ""

// Env returns the env
func Env() Envroinment {
	return Envroinment(env)
}

// Root returns the root path of oncekv
func Root() string {
	return root
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
	AdminAddr     string

	RaftKey string
}

func newDefaultConfig() Configuration {
	return Configuration{
		LogOut:                  os.Stdout,
		EtcdEndpoints:           []string{"127.0.0.1:2379"},
		OncekvMetaRefreshPeroid: time.Second,
		HTTPRequestTimeout:      time.Millisecond * 100,
		IdealResponseDuration:   time.Millisecond * 50,
		CacheBytes:              1 << 20,
		CacheMasterAddr:         "127.0.0.1:5550",
		RaftNodesKey:            "oncekv.db.nodes",
		CacheNodesKey:           "oncekv.cache.nodes",
		RaftKey:                 "oncekv.nodes.http.adrr",
		AdminAddr:               "127.0.0.1:5546",
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
			CacheMasterAddr:         "127.0.0.1:5550",
			RaftNodesKey:            "oncekv.db.nodes",
			CacheNodesKey:           "oncekv.cache.nodes",
			RaftKey:                 "oncekv.nodes.http.adrr",
			AdminAddr:               "127.0.0.1:5546",
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

	if r := os.Getenv("GOPATH"); r != "" {
		root = path.Join(r, "src", "github.com", "Focinfi", "oncekv")
	} else {
		panic("oncekv: envroinment param $GOPATH not set")
	}
}
