package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

var (
	log logger.Logger
)

// parseVector parses a comma-separated string of floats into a vector
func parseVector(s string) ([]float32, error) {
	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid vector component at position %d: %v", i, err)
		}
		vec[i] = float32(v)
	}
	return vec, nil
}

func parseMetadata(s string) (map[string]interface{}, error) {
	if s == "" {
		return nil, nil
	}

	metadata := make(map[string]interface{})
	pairs := strings.Split(s, ",")

	for _, pair := range pairs {
		kv := strings.Split(strings.TrimSpace(pair), "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid metadata format: %s", pair)
		}
		metadata[kv[0]] = kv[1]
	}

	return metadata, nil
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	searchVector := flag.String("search", "", "comma-separated vector to search for (e.g., '0.1,0.2,0.3')")
	insertVector := flag.String("insert", "", "comma-separated vector to insert")
	vectorID := flag.String("id", "", "ID for the vector to insert")
	metadata := flag.String("metadata", "", "metadata for the vector (format: key1=value1,key2=value2)")
	k := flag.Int("k", 5, "number of nearest neighbors to return")
	threshold := flag.Float64("threshold", 0.0, "similarity threshold (0.0 to 1.0)")
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
	eng := engine.NewEngine(cfg, log)

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

	// If insert vector is provided, perform insert
	if *insertVector != "" {
		if *vectorID == "" {
			log.Fatal("Vector ID is required for insertion")
		}

		vec, err := parseVector(*insertVector)
		if err != nil {
			log.Fatal("Failed to parse insert vector", logger.Error("error", err))
		}

		meta, err := parseMetadata(*metadata)
		if err != nil {
			log.Fatal("Failed to parse metadata", logger.Error("error", err))
		}

		err = eng.Insert(types.Vector{
			ID:        *vectorID,
			Embedding: vec,
			Metadata:  meta,
		})

		if err != nil {
			log.Error("Insert failed", logger.Error("error", err))
		} else {
			log.Info("Vector inserted successfully",
				logger.String("id", *vectorID),
				logger.Int("dimensions", len(vec)))
		}
		return
	}

	// If search vector is provided, perform search
	if *searchVector != "" {
		vec, err := parseVector(*searchVector)
		if err != nil {
			log.Fatal("Failed to parse search vector", logger.Error("error", err))
		}

		// Perform search
		results, err := eng.Search(types.Vector{
			Embedding: vec,
		}, engine.SearchParams{
			K:           *k,
			Threshold:   float32(*threshold),
			IncludeVecs: true,
			IncludeMeta: true,
		})

		if err != nil {
			log.Error("Search failed", logger.Error("error", err))
		} else {
			fmt.Println("\nSearch Results:")
			fmt.Println("---------------")
			for i, result := range results {
				fmt.Printf("%d. ID: %s, Score: %.4f\n", i+1, result.Vector.ID, result.Score)
				if result.Vector.Metadata != nil {
					fmt.Printf("   Metadata: %v\n", result.Vector.Metadata)
				}
			}
		}
		return
	}

	<-ctx.Done()
	log.Info("Shutting down...")

	// Cleanup
	if err := eng.Stop(); err != nil {
		log.Error("Error during shutdown", logger.Error("error", err))
	}
}
