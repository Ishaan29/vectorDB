package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/ishaan29/vectorDB/internal/config"
	"github.com/ishaan29/vectorDB/pkg/types"
)

// Engine represents the core vector database engine
type Engine struct {
	mu     sync.RWMutex
	config *config.Config
	store  map[string]types.Vector
	// TODO: Add index structure
	running bool
}

// NewEngine creates a new instance of the vector database engine
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		config: cfg,
		store:  make(map[string]types.Vector),
	}
}

// Start initializes and starts the engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine is already running")
	}

	// TODO: Initialize index
	// TODO: Load persisted data

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

	// TODO: Persist data
	// TODO: Clean up resources

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

	// TODO: Validate vector dimensions
	// TODO: Update index

	e.store[vector.ID] = vector
	return nil
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
