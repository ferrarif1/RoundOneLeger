package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ledger/internal/auth"
	"ledger/internal/models"
)

const (
	// ContextSessionKey stores the authenticated session on the Gin context.
	ContextSessionKey = "ledger/session"
)

// IPAllowlist enforces allowlist membership using the provided store.
func IPAllowlist(store *models.LedgerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.Next()
			return
		}
		ip := clientIP(c.Request)
		if ip == "" || !store.IsIPAllowed(ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "ip_not_allowed"})
			return
		}
		c.Next()
	}
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		candidate := strings.TrimSpace(parts[0])
		if ip := net.ParseIP(candidate); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RequireSession validates that a session token is present and valid.
func RequireSession(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try to get token from Authorization header
		token := c.GetHeader("Authorization")
		if token != "" {
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}
		} else {
			// Fallback to cookie
			cookie, err := c.Request.Cookie("ledger.session")
			if err != nil || cookie == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_session"})
				return
			}
			token = cookie.Value
		}
		
		session, ok := manager.Validate(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
			return
		}
		c.Set(ContextSessionKey, session)
		c.Next()
	}
}

// CORS adds permissive CORS headers to all responses to support requests
// served from a different origin. It mirrors the Origin header to support
// credentialed requests and terminates preflight checks early.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}