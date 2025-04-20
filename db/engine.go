package db

import (
	"sync"

	"github.com/ishaan29/vectorDB/storage"
)

type Engine struct {
	mu    sync.RWMutex
	store map[string]storage.Vector
	// index
}

func NewEngine() *Engine {
	return &Engine{
		store: make(map[string]storage.Vector),
	}
}

func (engine *Engine) Insert(vector storage.Vector) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.store[vector.ID] = vector
}

func (engine *Engine) Get(id string) (storage.Vector, bool) {
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	v, ok := engine.store[id]
	return v, ok
}
