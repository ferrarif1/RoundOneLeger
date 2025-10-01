package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ledger/internal/api"
	"ledger/internal/auth"
	"ledger/internal/db"
	"ledger/internal/models"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.ConnectFromEnv(ctx)
	if err != nil {
		log.Printf("database connection warning: %v", err)
	}

	if database != nil {
		defer func() {
			if err := database.Close(); err != nil {
				log.Printf("database close error: %v", err)
			}
		}()
	}

	fingerprintSecret := []byte(os.Getenv("FINGERPRINT_SECRET"))
	store := models.NewLedgerStore(fingerprintSecret)
	sessions := auth.NewManager(12 * time.Hour)

	router := api.NewRouter(api.Config{Database: database, Store: store, Sessions: sessions})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
