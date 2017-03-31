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

var (
	defaultAddr = config.Config.AdminAddr
	wsUpgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

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
	engine.GET("/caches", a.handleCaches)
	engine.GET("/dbs", a.handleDBs)
	engine.GET("/ws/caches", a.handleWebSocketCaches)
	engine.GET("/ws/dbs", a.handleWebSocketDBs)
	return engine
}

func (a *Admin) handleCaches(ctx *gin.Context) {
	peers, err := a.CacheMaster.Peers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	ctx.JSON(http.StatusOK, peers)
}

func (a *Admin) handleDBs(ctx *gin.Context) {
	peers, err := a.DBMaster.Peers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	ctx.JSON(http.StatusOK, peers)
}

func (a *Admin) handleWebSocketCaches(ctx *gin.Context) {
	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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
}

func (a *Admin) handleWebSocketDBs(ctx *gin.Context) {
	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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
}

// Default act as a default Admin
var Default *Admin

func init() {
	adm := &Admin{
		CacheMaster: cache.Default,
		DBMaster:    db.Default,
		addr:        defaultAddr,
	}

	adm.Engine = adm.newServer()
	Default = adm
}
