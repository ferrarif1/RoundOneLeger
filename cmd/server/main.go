package main

import (
	"context"
	"flag"
	"fmt"
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

	// Flags override env
	var (
		flagDataDir      = flag.String("data-dir", "", "Directory to persist snapshots")
		flagAutosaveSecs = flag.Int("autosave-secs", 0, "Autosave interval seconds (0 to disable)")
		flagRetention    = flag.Int("retention", 0, "Number of rolling backups to retain")
	)
	flag.Parse()

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

	store := models.NewLedgerStore()

	dataDir := *flagDataDir
	if dataDir == "" {
		dataDir = os.Getenv("LEDGER_DATA_DIR")
	}
	useDB := database != nil && database.SQL != nil
	retention := *flagRetention
	if retention <= 0 {
		if v := os.Getenv("LEDGER_SNAPSHOT_RETENTION"); v != "" {
			var parsed int
			if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
				retention = parsed
			}
		}
	}
	if retention <= 0 {
		retention = 10
	}

	if useDB {
		shouldPersist := false
		loaded, err := store.LoadFromDatabase(database.SQL)
		if err != nil {
			log.Printf("load snapshot from database error: %v", err)
			shouldPersist = true
		}
		if !loaded {
			shouldPersist = true
		}
		if shouldPersist {
			if err := store.SaveToDatabaseWithRetention(database.SQL, retention); err != nil {
				log.Printf("seed snapshot to database error: %v", err)
			} else {
				log.Printf("seeded default snapshot (with admin) to database")
			}
		}
	} else if dataDir != "" {
		if err := store.LoadFrom(dataDir); err != nil {
			log.Printf("load snapshot error: %v", err)
		}
	}

	// autosave ticker (database preferred, otherwise filesystem if configured)
	autosaveSecs := *flagAutosaveSecs
	if autosaveSecs <= 0 {
		if v := os.Getenv("LEDGER_AUTOSAVE_SECS"); v != "" {
			if n, err := time.ParseDuration(v + "s"); err == nil {
				autosaveSecs = int(n.Seconds())
			}
		}
	}
	if autosaveSecs <= 0 {
		autosaveSecs = 10
	}
	if useDB || dataDir != "" {
		autosave := time.NewTicker(time.Duration(autosaveSecs) * time.Second)
		defer autosave.Stop()
		go func() {
			for range autosave.C {
				retention := *flagRetention
				if retention <= 0 {
					if v := os.Getenv("LEDGER_SNAPSHOT_RETENTION"); v != "" {
						if n, err := time.ParseDuration("0s"); err == nil {
							_ = n
						}
					}
					if v := os.Getenv("LEDGER_SNAPSHOT_RETENTION"); v != "" {
						// simple parse int
					}
				}
				if retention <= 0 {
					retention = 10
				}
				if useDB {
					if err := store.SaveToDatabaseWithRetention(database.SQL, retention); err != nil {
						log.Printf("autosave db error: %v", err)
					}
					continue
				}
				if err := store.SaveToWithRetention(dataDir, retention); err != nil {
					log.Printf("autosave error: %v", err)
				}
			}
		}()
	}

	sessions := auth.NewManager(12 * time.Hour)

	router := api.NewRouter(api.Config{Database: database, Store: store, Sessions: sessions, DataDir: dataDir, Retention: retention})

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

	if useDB {
		if err := store.SaveToDatabaseWithRetention(database.SQL, retention); err != nil {
			log.Printf("final db save error: %v", err)
		}
	} else if dataDir != "" {
		if err := store.SaveToWithRetention(dataDir, retention); err != nil {
			log.Printf("final save error: %v", err)
		}
	}
}
