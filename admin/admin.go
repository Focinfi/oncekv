package admin

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	cache "github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/config"
	db "github.com/Focinfi/oncekv/db/master"
	"github.com/Focinfi/sqs/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var defaulAddr = config.Config.AdminAddr

// Admin for oncekv admin
type Admin struct {
	addr string
	*gin.Engine

	CacheMaster *cache.Master
	DBMaster    *db.Master
}

// Start starts the admin server
func (a *Admin) Start() {
	go a.DBMaster.Start()
	a.Run(a.addr)
}

func (a *Admin) newServer() *gin.Engine {
	engine := gin.Default()
	engine.GET("/caches", func(ctx *gin.Context) {
		peers, err := a.CacheMaster.Peers()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		ctx.JSON(http.StatusOK, peers)
	})

	engine.GET("/dbs", func(ctx *gin.Context) {
		peers, err := a.DBMaster.Peers()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		ctx.JSON(http.StatusOK, peers)
	})

	var wsupgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	engine.GET("/ws/caches", func(ctx *gin.Context) {
		conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Internal.Error(err)
			return
		}
		defer conn.Close()

		peers := []string{}

		for {
			select {
			case <-time.After(time.Second):
				newPeers, err := a.CacheMaster.Peers()
				if err != nil {
					log.DB.Error(err)
					continue
				}

				if reflect.DeepEqual(newPeers, peers) {
					continue
				}

				b, err := json.Marshal(newPeers)
				if err != nil {
					log.DB.Error(err)
					continue
				}

				peers = newPeers
				conn.WriteMessage(1, b)
			}
		}
	})

	engine.GET("/ws/dbs", func(ctx *gin.Context) {
		conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Internal.Error(err)
			return
		}
		defer conn.Close()

		peers := []string{}

		for {
			select {
			case <-time.After(time.Second):
				newPeers, err := a.DBMaster.Peers()
				if err != nil {
					log.DB.Error(err)
					continue
				}
				log.DB.Infoln("newPeers", newPeers)

				if reflect.DeepEqual(newPeers, peers) {
					continue
				}

				b, err := json.Marshal(newPeers)
				if err != nil {
					log.DB.Error(err)
					continue
				}

				peers = newPeers
				if err := conn.WriteMessage(1, b); err != nil {
					return
				}
			}
		}
	})

	return engine
}

// Default act as a default Admin
var Default *Admin

func init() {
	admin := &Admin{
		CacheMaster: cache.Default,
		DBMaster:    db.Default,
		addr:        defaulAddr,
	}

	admin.Engine = admin.newServer()
	Default = admin
}
