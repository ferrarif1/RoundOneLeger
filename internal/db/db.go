package db

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type Database struct {
	cfg Config
}

func ConnectFromEnv(ctx context.Context) (*Database, error) {
	database := &Database{cfg: loadConfigFromEnv()}
	if err := database.PingContext(ctx); err != nil {
		return database, fmt.Errorf("database ping failed: %w", err)
	}
	return database, nil
}

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

func (d *Database) PingContext(ctx context.Context) error {
	if d == nil {
		return fmt.Errorf("database is not initialized")
	}
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(d.cfg.Host, d.cfg.Port))
	if err != nil {
		return err
	}
	return conn.Close()
}

func (d *Database) Close() error {
	return nil
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
