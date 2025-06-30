package main

import (
	"context"
	"log"
	"mtracker/internal/config"
	"mtracker/internal/db"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// initalize db
	database, err := db.NewConnection(cfg.DatabaseURL.URL)
	if err != nil {
		log.Fatalf("Failed to initalize database: %v", err)
	}
	defer database.Close()

	// migrations

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}
	// goroutine start server
	go func() {
		log.Printf("server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
