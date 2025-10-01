package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ledger/internal/db"
)

// NewRouter configures HTTP routes for the application.
func NewRouter(database *db.Database) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		if database != nil {
			if err := database.PingContext(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}
