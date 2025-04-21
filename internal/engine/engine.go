package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

// Engine represents the core vector database engine
type Engine struct {
	mu     sync.RWMutex
	config *config.Config
	store  map[string]types.Vector
	logger logger.Logger
	// TODO: Add index structure
	running bool
}

// NewEngine creates a new instance of the vector database engine
func NewEngine(cfg *config.Config, log logger.Logger) *Engine {
	return &Engine{
		config: cfg,
		store:  make(map[string]types.Vector),
		logger: log,
	}
}

// Start initializes and starts the engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine is already running")
	}

	// Load persisted data
	if err := e.loadVectors(); err != nil {
		e.logger.Warn("Failed to load vectors", logger.Error("error", err))
	}

	e.running = true
	return nil
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("engine is not running")
	}

	// Persist data before stopping
	if err := e.saveVectors(); err != nil {
		e.logger.Error("Failed to save vectors", logger.Error("error", err))
	}

	e.running = false
	return nil
}

// Insert adds a new vector to the database
func (e *Engine) Insert(vector types.Vector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("engine is not running")
	}

	if len(vector.Embedding) != e.config.Index.Dimensions {
		return fmt.Errorf("invalid vector dimensions: expected %d, got %d",
			e.config.Index.Dimensions, len(vector.Embedding))
	}

	e.store[vector.ID] = vector

	// Save after each insert for now (can be optimized later)
	return e.saveVectors()
}

// Query performs a similarity search
func (e *Engine) Query(query types.Vector, k int) ([]types.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, fmt.Errorf("engine is not running")
	}

	// TODO: Implement similarity search
	// For now, return empty results
	return []types.SearchResult{}, nil
}

// Get retrieves a vector by ID
func (e *Engine) Get(id string) (types.Vector, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	v, ok := e.store[id]
	return v, ok
}

// Delete removes a vector from the database
func (e *Engine) Delete(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("engine is not running")
	}

	// TODO: Update index
	delete(e.store, id)
	return nil
}

func (e *Engine) loadVectors() error {
	path := filepath.Join(e.config.Storage.Path, "vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data file yet
		}
		return err
	}

	return json.Unmarshal(data, &e.store)
}

func (e *Engine) saveVectors() error {
	if err := os.MkdirAll(e.config.Storage.Path, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(e.store)
	if err != nil {
		return err
	}

	path := filepath.Join(e.config.Storage.Path, "vectors.json")
	return os.WriteFile(path, data, 0644)
}
