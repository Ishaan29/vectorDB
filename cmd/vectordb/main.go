package main

import (
	"context"
	"flag"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
)

var (
	log logger.Logger
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Initialize temporary logger for startup
	tmpLogger := stdlog.New(os.Stdout, "[vectordb] ", stdlog.LstdFlags)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		tmpLogger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	log, err = logger.New(&cfg.Logging)
	if err != nil {
		tmpLogger.Fatalf("Failed to initialize logger: %v", err)
	}
	defer log.Sync()

	log.Info("Starting VectorDB",
		logger.String("config_path", *configPath),
		logger.String("version", "0.1.0"),
	)

	// Create engine instance
	eng := engine.NewEngine(cfg)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received signal", logger.String("signal", sig.String()))
		cancel()
	}()

	// Start the engine
	if err := eng.Start(ctx); err != nil {
		log.Fatal("Failed to start engine", logger.Error("error", err))
	}

	<-ctx.Done()
	log.Info("Shutting down...")

	// Cleanup
	if err := eng.Stop(); err != nil {
		log.Error("Error during shutdown", logger.Error("error", err))
	}
}
