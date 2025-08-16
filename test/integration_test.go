package test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/engine"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

func TestCompleteFlow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			Path: tempDir,
		},
		Index: config.IndexConfig{
			Type:       "hnsw",
			Dimensions: 128,
		},
		Badger: config.BadgerConfig{
			Path: tempDir,
		},
	}

	log, _ := logger.New(&logger.Config{
		Level:       "debug",
		Encoding:    "json",
		OutputPaths: []string{"stdout"},
	})

	ctx := context.Background()

	// Test 1: Create and start engine
	t.Run("StartEngine", func(t *testing.T) {
		eng, err := engine.NewEngine(cfg, log)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		err = eng.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start engine: %v", err)
		}

		defer eng.Stop()
	})

	// Test 2: Insert and search
	t.Run("InsertAndSearch", func(t *testing.T) {
		eng, _ := engine.NewEngine(cfg, log)
		eng.Start(ctx)
		defer eng.Stop()

		// Insert vectors
		vectors := make([]types.Vector, 10)
		for i := 0; i < 10; i++ {
			vectors[i] = types.Vector{
				ID:        fmt.Sprintf("vec%d", i),
				Embedding: generateRandomVector(128),
				Metadata:  map[string]interface{}{"index": i},
			}
		}

		for _, v := range vectors {
			if err := eng.Insert(v); err != nil {
				t.Errorf("Failed to insert: %v", err)
			}
		}

		// Search
		results, err := eng.Search(vectors[0], engine.SearchParams{
			K:           3,
			Threshold:   0.0,
			IncludeVecs: false,
			IncludeMeta: true,
		})

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("No results")
		}

		if results[0].Vector.ID != "vec0" {
			t.Errorf("Expected vec0, got %s", results[0].Vector.ID)
		}
	})

	// Test 3: Persistence
	t.Run("Persistence", func(t *testing.T) {
		// First engine - insert data
		eng1, _ := engine.NewEngine(cfg, log)
		eng1.Start(ctx)

		testVec := types.Vector{
			ID:        "persistent",
			Embedding: generateRandomVector(128),
			Metadata:  map[string]interface{}{"test": true},
		}

		eng1.Insert(testVec)
		eng1.Stop()

		// Second engine - verify data persisted
		eng2, _ := engine.NewEngine(cfg, log)
		eng2.Start(ctx)
		defer eng2.Stop()

		vec, found := eng2.Get("persistent")
		if !found {
			t.Fatal("Vector not persisted")
		}

		if vec.ID != "persistent" {
			t.Errorf("Wrong vector retrieved")
		}
	})
}

func generateRandomVector(dim int) []float32 {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = rand.Float32()
	}
	return vec
}
