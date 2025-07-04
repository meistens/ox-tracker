package main

import (
	"context"
	"flag"
	"log"
	"mtracker/internal/bot/telegram"
	"mtracker/internal/commands"
	"mtracker/internal/config"
	"mtracker/internal/db"
	"mtracker/seed"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// application struct
type application struct {
	config config.Config
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// use flags to read values for ports and environment
	// default to using set parameters if no values is passed
	flag.IntVar(&cfg.Server.Port, "Port", 5000, "App server Port") // test to see which one returns what port
	flag.StringVar(&cfg.Env, "Env", "development", "Environment (dev|stage|prod... all in full letters)")
	flag.Parse()

	// initalize db
	database, err := db.NewConnection(cfg.DatabaseURL.URL)
	if err != nil {
		log.Fatalf("Failed to initalize database: %v", err)
	}
	defer database.Close()

	//seed
	seed.SeedMediaFromJSON(database, "./seed/media_seed.json")
	// seed
	app := &application{
		config: *cfg,
	}

	// TODO: instance of...
	mux := http.NewServeMux()
	addr := cfg.Server.Port
	mux.HandleFunc("/v1/health", app.healthCheckHandler)

	// Initialize command handler and telegram handler
	mediaRepo := db.NewMediaRepository(database)
	userMediaRepo := db.NewUserMediaRepository(database)
	userRepo := db.NewUserRepository(database)
	cmdHandler := commands.NewCommandHandler(mediaRepo, userMediaRepo, userRepo)
	tgHandler := telegram.NewTelegramHandler(cfg.BotTokens.TelegramToken, cmdHandler)

	// --- Telegram Bot Startup (polling mode for local development) ---
	go func() {
		if err := tgHandler.Start(); err != nil {
			log.Printf("Telegram bot error: %v", err)
		}
	}()
	log.Println("Telegram bot running in polling mode")
	// --- End Telegram Bot Startup ---

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(addr),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	// goroutine start server
	go func() {
		log.Printf("starting %s server starting on port %d", cfg.Env, cfg.Server.Port)
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
