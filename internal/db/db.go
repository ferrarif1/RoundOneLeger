package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	SQL *sql.DB
}

func (d *Database) Config() Config {
	return d.cfg
}

func ConnectFromEnv(ctx context.Context) (*Database, error) {
	cfg := loadConfigFromEnv()
	database := &Database{cfg: cfg}

	// Open SQL connection early so we can reuse it for persistence.
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return database, fmt.Errorf("database open failed: %w", err)
	}
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return database, fmt.Errorf("database ping failed: %w", err)
	}
	database.SQL = sqlDB

	// Keep TCP reachability check for parity with previous healthcheck.
	if err := database.PingContext(ctx); err != nil {
		return database, fmt.Errorf("database tcp check failed: %w", err)
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
	if d.SQL != nil {
		return d.SQL.PingContext(ctx)
	}
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(d.cfg.Host, d.cfg.Port))
	if err != nil {
		return err
	}
	return conn.Close()
}

func (d *Database) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

func loadConfigFromEnv() Config {
	return Config{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenv("DB_PORT", "5433"),
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
