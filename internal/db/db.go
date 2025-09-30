package db

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Config holds database configuration loaded from the environment.
type Config struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// Database represents a lightweight connection manager for PostgreSQL.
type Database struct {
	cfg Config

	mu       sync.RWMutex
	lastPing time.Time
	lastErr  error
}

// ConnectFromEnv reads configuration from environment variables and returns a
// Database handle. The handle is provided even when the initial ping fails so
// callers can continue operating and surface the error gracefully.
func ConnectFromEnv(ctx context.Context) (*Database, error) {
	database := &Database{cfg: loadConfigFromEnv()}
	if err := database.PingContext(ctx); err != nil {
		return database, fmt.Errorf("database ping failed: %w", err)
	}
	return database, nil
}

// DSN builds a PostgreSQL connection string from the configuration.
func (d *Database) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		d.cfg.Host,
		d.cfg.Port,
		d.cfg.Name,
		d.cfg.User,
		d.cfg.Password,
	)
}

// PingContext attempts to open a TCP connection to the configured PostgreSQL
// host and records the outcome for introspection.
func (d *Database) PingContext(ctx context.Context) error {
	if d == nil {
		return errors.New("database is not initialized")
	}

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(d.cfg.Host, d.cfg.Port))

	d.mu.Lock()
	d.lastPing = time.Now()
	d.lastErr = err
	d.mu.Unlock()

	if err != nil {
		return err
	}

	return conn.Close()
}

// Close exists for symmetry with real database connectors.
func (d *Database) Close() error {
	return nil
}

// LastStatus returns the timestamp and error from the most recent ping.
func (d *Database) LastStatus() (time.Time, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastPing, d.lastErr
}

func loadConfigFromEnv() Config {
	return Config{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenv("DB_PORT", "5432"),
		Name:     getenv("DB_NAME", "ledger"),
		User:     getenv("DB_USER", "postgres"),
		Password: getenv("DB_PASS", "postgres"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
