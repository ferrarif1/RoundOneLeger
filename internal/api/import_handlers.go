package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"ledger/internal/models"
)

func (s *Server) registerImportRoutes(group *gin.RouterGroup) {
	if s.Import == nil || s.Database == nil || s.Database.SQL == nil {
		group.POST("/import", func(c *gin.Context) { c.Status(http.StatusNotImplemented) })
		return
	}

	group.POST("/import", func(c *gin.Context) {
		tableName := c.Request.FormValue("tableName")
		_, fileHeader, err := c.Request.FormFile("file")
		if err != nil || fileHeader == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file_required"})
			return
		}
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unsupported_file"})
			return
		}
		if fileHeader.Size > 50*1024*1024 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file_too_large"})
			return
		}
		// TODO: validate size/type
		dstDir := os.TempDir()
		dstPath := filepath.Join(dstDir, "import-"+fileHeader.Filename)
		if err := saveUploadedFile(fileHeader, dstPath); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "save_failed"})
			return
		}
		// Create table first
		table, err := s.Roleger.CreateTable(c.Request.Context(), models.Table{
			Name: tableName,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		task, err := s.Import.CreateTask(c.Request.Context(), table.ID, dstPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Kick off async import (simplified placeholder)
		go func() {
			_ = s.Import.UpdateTaskStatus(context.Background(), task.ID, models.ImportRunning, 0, "")
			var err error
			if ext == ".csv" {
				err = s.Import.ProcessCSV(context.Background(), task)
			} else {
				err = s.Import.ProcessXLSX(context.Background(), task)
			}
			if err != nil {
				_ = s.Import.UpdateTaskStatus(context.Background(), task.ID, models.ImportFailed, 0, err.Error())
				return
			}
			s.Import.Cleanup(task)
		}()

		c.JSON(http.StatusOK, gin.H{"taskId": task.ID, "tableId": table.ID})
	})

	group.GET("/import/:id", func(c *gin.Context) {
		task, err := s.Import.GetTask(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task_not_found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"task": task})
	})
}
