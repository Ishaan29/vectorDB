package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(&cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Create and start engine
	eng, err := engine.NewEngine(cfg, log)
	if err != nil {
		log.Fatal("Failed to create engine", logger.Error("error", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Shutdown signal received")
		cancel()
	}()

	// Start engine (loads vectors from BadgerDB into HNSW)
	if err := eng.Start(ctx); err != nil {
		log.Fatal("Failed to start engine", logger.Error("error", err))
	}

	// Example: Insert some vectors
	testVectors := []types.Vector{
		{
			ID:        "doc1",
			Embedding: generateEmbedding(cfg.Index.Dimensions),
			Metadata:  map[string]interface{}{"title": "First Document"},
		},
		{
			ID:        "doc2",
			Embedding: generateEmbedding(cfg.Index.Dimensions),
			Metadata:  map[string]interface{}{"title": "Second Document"},
		},
	}

	for _, v := range testVectors {
		if err := eng.Insert(v); err != nil {
			log.Error("Failed to insert", logger.Error("error", err))
		}
	}

	// Example: Search
	query := types.Vector{
		Embedding: testVectors[0].Embedding, // Search with first vector
	}

	results, err := eng.Search(query, engine.SearchParams{
		K:           5,
		Threshold:   0.5,
		IncludeVecs: false,
		IncludeMeta: true,
	})

	if err != nil {
		log.Error("Search failed", logger.Error("error", err))
	} else {
		for _, r := range results {
			fmt.Printf("Found: %s (score: %.3f)\n", r.Vector.ID, r.Score)
		}
	}

	// Wait for shutdown
	<-ctx.Done()

	// Cleanup
	if err := eng.Stop(); err != nil {
		log.Error("Failed to stop engine", logger.Error("error", err))
	}
}

func generateEmbedding(dim int) []float32 {
	// Generate random embedding for testing
	emb := make([]float32, dim)
	for i := range emb {
		emb[i] = float32(i) / float32(dim)
	}
	return emb
}
