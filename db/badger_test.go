package db

import (
	"encoding/json"
	"testing"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/persistence"
	"github.com/ishaan29/vectorDB/pkg/types"
)

// TestBadgerDBConnection tests basic BadgerDB connectivity

func TestBadgerDBConnection(t *testing.T) {
	// Create a temporary directory for the test database
	cfg, err := config.Load("../config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg.Logging.Level = "debug"
	log, err := logger.New(&cfg.Logging)
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer log.Sync()

	store, err := persistence.NewBadgerStore(cfg.Badger.Path, log)
	if err != nil {
		log.Error("Failed to create Badger store", logger.Error("error", err))
		t.Fatalf("Failed to create Badger store: %v", err)
	}

	vector := types.Vector{
		ID:        "1",
		Embedding: []float32{0.1, 0.2, 0.3},
		Metadata:  map[string]interface{}{"name": "test"},
	}

	// test put

	err = store.Put(vector)
	if err != nil {
		log.Error("Failed to put vector", logger.Error("error", err))
		t.Fatalf("Failed to put vector: %v", err)
	}

	vector, err = store.Get("1")
	if err != nil {
		log.Error("Failed to get vector", logger.Error("error", err))
		t.Fatalf("Failed to get vector: %v", err)
	}

	if b, mErr := json.Marshal(vector); mErr != nil {
		log.Error("Failed to marshal vector", logger.Error("error", mErr))
	} else {
		log.Info("Vector", logger.String("vector", string(b)))
	}
}
