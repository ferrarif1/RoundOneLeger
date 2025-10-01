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

// RequireSession ensures a valid session token is provided via the Authorization header.
func RequireSession(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		session, ok := manager.Validate(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		c.Set(ContextSessionKey, session)
		c.Next()
	}
}

// OptionalSession attaches the session to the context if present but does not enforce authentication.
func OptionalSession(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.Next()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if session, ok := manager.Validate(token); ok {
			c.Set(ContextSessionKey, session)
		}
		c.Next()
	}
}
