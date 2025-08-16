package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ishaan29/vectorDB/internal/api"
	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	log_instance, err := logger.New(&cfg.Logging)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer log_instance.Sync()

	log_instance.Info("Starting VectorDB HTTP Server",
		logger.String("config", configPath),
		logger.String("host", cfg.Server.Host),
		logger.Int("port", cfg.Server.Port))

	// Initialize engine
	eng, err := engine.NewEngine(cfg, log_instance)
	if err != nil {
		log_instance.Fatal("Failed to create engine", logger.Error("error", err))
	}

	// Start engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eng.Start(ctx); err != nil {
		log_instance.Fatal("Failed to start engine", logger.Error("error", err))
	}
	defer func() {
		if err := eng.Stop(); err != nil {
			log_instance.Error("Failed to stop engine", logger.Error("error", err))
		}
	}()

	log_instance.Info("Engine started successfully")

	// Create and start HTTP server
	server := api.NewServer(eng, log_instance, cfg)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverCtx, serverCancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Start(serverCtx); err != nil {
			log_instance.Error("HTTP server error", logger.Error("error", err))
		}
	}()

	// Wait for shutdown signal
	<-quit
	log_instance.Info("Shutdown signal received")

	// Cancel server context to trigger graceful shutdown
	serverCancel()

	log_instance.Info("VectorDB HTTP Server stopped")
}
