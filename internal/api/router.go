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
		payload := gin.H{"status": "ok"}

		if database != nil {
			if err := database.PingContext(c.Request.Context()); err != nil {
				payload["database"] = gin.H{
					"status": "unavailable",
					"error":  err.Error(),
				}
			} else {
				payload["database"] = gin.H{"status": "ok"}
			}
		}

		c.JSON(http.StatusOK, payload)
	})

	return r
}
