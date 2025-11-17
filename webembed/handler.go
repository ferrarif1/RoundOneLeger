package webembed

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var embeddedDist embed.FS

var spaFS fs.FS

func init() {
	var err error
	spaFS, err = fs.Sub(embeddedDist, "dist")
	if err != nil {
		spaFS = nil
	}
}

// Register mounts the embedded frontend on the provided router.
func Register(router *gin.Engine) bool {
	handler := spaHandler{fs: spaFS}
	router.GET("/", handler.Serve)
	router.GET("/index.html", handler.Serve)
	router.GET("/assets/*filepath", handler.Serve)
	router.GET("/*filepath", handler.Serve)
	return handler.fs != nil
}

type spaHandler struct {
	fs fs.FS
}

func (h spaHandler) Serve(c *gin.Context) {
	if shouldBypass(c.Request.URL.Path) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if h.fs == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "frontend_not_built"})
		return
	}

	requestPath := normalizePath(c.Request.URL.Path)
	data, name, err := h.readFile(requestPath)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "frontend_unavailable"})
		return
	}

	reader := bytes.NewReader(data)
	http.ServeContent(c.Writer, c.Request, name, time.Time{}, reader)
}

func (h spaHandler) readFile(requestPath string) ([]byte, string, error) {
	if requestPath == "" {
		requestPath = "index.html"
	}

	data, err := fs.ReadFile(h.fs, requestPath)
	if err == nil {
		return data, requestPath, nil
	}

	data, err = fs.ReadFile(h.fs, "index.html")
	if err != nil {
		return nil, "", err
	}
	return data, "index.html", nil
}

func normalizePath(requestPath string) string {
	clean := strings.TrimPrefix(path.Clean(requestPath), "/")
	if clean == "" || strings.HasSuffix(requestPath, "/") {
		return "index.html"
	}
	return clean
}

func shouldBypass(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/api/") ||
		strings.HasPrefix(requestPath, "/auth/") ||
		requestPath == "/health"
}
