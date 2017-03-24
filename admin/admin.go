package admin

import (
	"net/http"

	cache "github.com/Focinfi/oncekv/cache/master"
	"github.com/Focinfi/oncekv/config"
	db "github.com/Focinfi/oncekv/db/master"
	"github.com/gin-gonic/gin"
)

var defaulAddr = config.Config().AdminAddr

// Admin for oncekv admin
type Admin struct {
	addr string
	*gin.Engine

	CacheMaster *cache.Master
	DBMaster    *db.Master
}

// Start starts the admin server
func (a *Admin) Start() {
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
