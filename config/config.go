package config

import (
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
	return e == develop
}

var env = Envroinment(develop)

// Env returns the env
func Env() Envroinment {
	return env
}

// Configuration defines configuration
type Configuration struct {
	LogOut                  io.Writer
	EtcdEndpoints           []string
	OncekvMetaRefreshPeroid time.Duration
	ClientRequestTimeout    time.Duration
	IdealResponseDuration   time.Duration
}

func newDefaultConfig() Configuration {
	return Configuration{
		LogOut:                  os.Stdout,
		EtcdEndpoints:           []string{"localhost:2379"},
		OncekvMetaRefreshPeroid: time.Second,
		ClientRequestTimeout:    time.Millisecond * 100,
		IdealResponseDuration:   time.Millisecond * 50,
	}
}

// Config returns the Configuration based on envroinment
func Config() Configuration {

	switch env {
	case productionEnv:
		return Configuration{
			LogOut:                  os.Stdout,
			OncekvMetaRefreshPeroid: time.Second,
			ClientRequestTimeout:    time.Millisecond * 100,
			IdealResponseDuration:   time.Millisecond * 50,
		}
	case developEnv:
		return newDefaultConfig()
	default:
		return newDefaultConfig()
	}
}

func init() {
	if e := os.Getenv("SQS_ENV"); e != "" {
		env = Envroinment(e)
	}

	if Env().IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
}
