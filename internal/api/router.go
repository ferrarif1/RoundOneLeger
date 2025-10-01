package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"ledger/internal/db"
	"ledger/internal/ledger"
)

// NewRouter configures the HTTP routes for the application.
func NewRouter(database *db.Database) *gin.Engine {
	router := gin.Default()
	ledgerStore := ledger.NewStore()
	ledgerHandler := &LedgerHandler{Store: ledgerStore}

	router.GET("/health", func(c *gin.Context) {
		if err := database.PingContext(c.Request.Context()); err != nil {
			log.Printf("database health check failed: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	ledgerHandler.RegisterRoutes(router)

	return router
}
