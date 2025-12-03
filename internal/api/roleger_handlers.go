package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ledger/internal/models"
)

// registerRolegerRoutes attaches Roleger-style DB endpoints when service available.
func (s *Server) registerRolegerRoutes(group *gin.RouterGroup) {
	if s.Roleger == nil {
		group.GET("/tables", func(c *gin.Context) { c.Status(http.StatusNotImplemented) })
		return
	}

	group.GET("/tables", func(c *gin.Context) {
		tables, err := s.Roleger.ListTables(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tables": tables})
	})

	group.POST("/tables", func(c *gin.Context) {
		var payload struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Name) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		table, err := s.Roleger.CreateTable(c.Request.Context(), models.Table{
			ID:          payload.ID,
			Name:        payload.Name,
			Description: payload.Description,
			CreatedBy:   currentSession(c, s.Sessions),
			UpdatedAt:   time.Now(),
			Version:     1,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"table": table})
	})

	group.GET("/tables/:id/views", func(c *gin.Context) {
		views, err := s.Roleger.ListViews(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"views": views})
	})

	group.POST("/tables/:id/views", func(c *gin.Context) {
		var payload models.View
		if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Name) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		payload.TableID = c.Param("id")
		if payload.TableID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "table_required"})
			return
		}
		view, err := s.Roleger.CreateView(c.Request.Context(), payload)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"view": view})
	})

	group.PUT("/tables/:id/views/:viewId", func(c *gin.Context) {
		var payload models.View
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		payload.TableID = c.Param("id")
		payload.ID = c.Param("viewId")
		view, err := s.Roleger.UpdateView(c.Request.Context(), payload)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"view": view})
	})

	group.GET("/tables/:id/properties", func(c *gin.Context) {
		propsProvider, ok := s.Roleger.(interface {
			ListProperties(ctx context.Context, tableID string) ([]models.Property, error)
		})
		if !ok {
			c.Status(http.StatusNotImplemented)
			return
		}
		properties, err := propsProvider.ListProperties(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"properties": properties})
	})

	group.POST("/tables/:id/properties", func(c *gin.Context) {
		propsProvider, ok := s.Roleger.(interface {
			CreateProperty(ctx context.Context, prop models.Property) (*models.Property, error)
		})
		if !ok {
			c.Status(http.StatusNotImplemented)
			return
		}
		var payload models.Property
		if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Name) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		payload.TableID = c.Param("id")
		created, err := propsProvider.CreateProperty(c.Request.Context(), payload)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"property": created})
	})

	group.GET("/tables/:id/records", func(c *gin.Context) {
		limit, offset := parsePagination(c, 100, 0)
		filters, sorts, ok := parseFilters(c)
		if !ok {
			return
		}
		records, total, err := s.Roleger.ListRecords(c.Request.Context(), c.Param("id"), limit, offset, filters, sorts)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records, "total": total, "limit": limit, "offset": offset})
	})

	type updateRecordPayload struct {
		Properties map[string]interface{} `json:"properties"`
	}

	group.PUT("/tables/:id/records/:recordId", func(c *gin.Context) {
		var payload updateRecordPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		record, err := s.Roleger.UpdateRecordProperties(c.Request.Context(), c.Param("id"), c.Param("recordId"), payload.Properties)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"record": record})
	})

	group.POST("/tables/:id/records/bulk", func(c *gin.Context) {
		var payload struct {
			Updates []struct {
				ID         string                 `json:"id"`
				Properties map[string]interface{} `json:"properties"`
			} `json:"updates"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		tableID := c.Param("id")
		var updated []models.RecordItem
		for _, upd := range payload.Updates {
			record, err := s.Roleger.UpdateRecordProperties(c.Request.Context(), tableID, upd.ID, upd.Properties)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "failed_id": upd.ID})
				return
			}
			updated = append(updated, *record)
		}
		c.JSON(http.StatusOK, gin.H{"records": updated})
	})

	group.POST("/tables/:id/records", func(c *gin.Context) {
		var payload updateRecordPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
			return
		}
		record, err := s.Roleger.CreateRecord(c.Request.Context(), c.Param("id"), payload.Properties, currentSession(c, s.Sessions))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"record": record})
	})

	group.DELETE("/tables/:id/records/:recordId", func(c *gin.Context) {
		if err := s.Roleger.DeleteRecord(c.Request.Context(), c.Param("id"), c.Param("recordId"), currentSession(c, s.Sessions)); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})
}

func parsePagination(c *gin.Context, defaultLimit, defaultOffset int) (int, int) {
	limit := defaultLimit
	offset := defaultOffset
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
