package config

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/jinzhu/configor"
)

var root string

// Root returns the root path of oncekv
func Root() string {
	return root
}

// Env for application environment
type Env string

// IsProduction returns if the env equals to production
func (e Env) IsProduction() bool {
	return e == "production"
}

// IsDevelop returns if the env equals to develop
func (e Env) IsDevelop() bool {
	return e == "develop"
}

// IsTest returns if the env equals to develop
func (e Env) IsTest() bool {
	return e == "test"
}

// Config for config
var Config = struct {
	Env  Env `default:"develop" env:"ONCEKV_ENV"`
	Root string

	HTTPRequestTimeout    time.Duration `default:"100000000" default:"ONCEKV_HTTP_REQUEST_TIMEOUT"`
	IdealResponseDuration time.Duration `default:"50000000" default:"ONCEKV_HTTP_IDEAL_RESPONSE_DURATION"`

	// etcd addrs and the the meta data key
	EtcdEndpoints []string `default:"['127.0.0.1:2379']" env:"ONCEKV_ETCD_ADDRS"`
	RaftKey       string   `default:"oncekv.nodes.http.adrr" env:"ONCEKV_DB_NODE_KEY"`

	RaftNodesKey  string `default:"oncekv.db.nodes" env:"ONCEKV_DB_NODES_KEY"`
	CacheNodesKey string `default:"oncekv.cache.nodes" env:"ONCEKV_CACHE_NODES_KEY"`

	// db shard master
	ShardCount int `default:"10" env:"ONCEKV_SHARD_COUNT"`

	// cache server
	CacheMasterAddr string `default:"127.0.0.1:5550" env:"ONCEKV_CACHE_MASTER_ADDR"`
	// default is 10M
	CacheBytes int64 `default:"10485760" env:"ONCEKV_CACHE_BYTES"`

	// admin
	AdminAddr string `default:"127.0.0.1:5546" env:"ONCEKV_ADMIN_ADDR"`

	// TODO: choose log collector
	LogOut io.Writer
}{}

func init() {
	if r := os.Getenv("GOPATH"); r != "" {
		root = path.Join(r, "src", "github.com", "Focinfi", "oncekv")
	} else {
		panic("oncekv: envroinment param $GOPATH not set")
	}

	err := configor.Load(&Config, path.Join(root, "config", "config.json"))
	if err != nil {
		panic(err)
	}

	fmt.Println(Config)
}
