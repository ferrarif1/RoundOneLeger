package api

import (
	"github.com/gin-gonic/gin"

	"ledger/internal/auth"
	"ledger/internal/db"
	"ledger/internal/middleware"
	"ledger/internal/models"
	"ledger/internal/services"
	"ledger/webembed"
)

// Config configures the router dependencies.
type Config struct {
	Database  *db.Database
	Store     *models.LedgerStore
	Sessions  *auth.Manager
	DataDir   string
	Retention int
	Roleger   *services.RolegerService
	Import    *services.ImportService
}

// NewRouter configures HTTP routes for the application.
func NewRouter(cfg Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS())

	server := &Server{
		Database:          cfg.Database,
		Store:             cfg.Store,
		Sessions:          cfg.Sessions,
		DataDir:           cfg.DataDir,
		SnapshotRetention: cfg.Retention,
		Roleger:           cfg.Roleger,
		Import:            cfg.Import,
	}
	server.RegisterRoutes(r)
	webembed.Register(r)

	return r
}
