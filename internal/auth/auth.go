package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Session represents an issued login token.
type Session struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ClientID  string    `json:"client_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Manager tracks active sessions in-memory.
type Manager struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]*Session
}

// NewManager constructs a session manager with the provided TTL.
func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &Manager{ttl: ttl, entries: make(map[string]*Session)}
}

// Issue creates a new session for a username and client fingerprint.
func (m *Manager) Issue(username, clientID string) (*Session, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session := &Session{
		Token:     token,
		Username:  username,
		ClientID:  clientID,
		IssuedAt:  now,
		ExpiresAt: now.Add(m.ttl),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[token] = session
	return session, nil
}

// Validate looks up a token and returns the session if still valid.
func (m *Manager) Validate(token string) (*Session, bool) {
	m.mu.RLock()
	session, ok := m.entries[token]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(session.ExpiresAt) {
		m.Revoke(token)
		return nil, false
	}
	return session, true
}

// Revoke removes a token from the manager.
func (m *Manager) Revoke(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, token)
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
