package db

import (
	"container/heap"
	"sync"

	"github.com/ishaan29/vectorDB/pkg/vectormath"
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

// SearchResult represents a single search result with its similarity score
type SearchResult struct {
	Vector     storage.Vector
	Similarity float32
}

// ResultHeap is a min-heap of SearchResults
type ResultHeap []SearchResult

func (h ResultHeap) Len() int            { return len(h) }
func (h ResultHeap) Less(i, j int) bool  { return h[i].Similarity < h[j].Similarity }
func (h ResultHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *ResultHeap) Push(x interface{}) { *h = append(*h, x.(SearchResult)) }
func (h *ResultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Search performs a K-nearest neighbor search using cosine similarity
func (engine *Engine) Search(query []float32, k int) ([]SearchResult, error) {
	engine.mu.RLock()
	defer engine.mu.RUnlock()

	// Initialize a min-heap to store top K results
	h := &ResultHeap{}
	heap.Init(h)

	// Compare query vector with all vectors in store
	for _, vector := range engine.store {
		similarity, err := vectormath.CosineSimilarity(query, vector.Embedding)
		if err != nil {
			continue // Skip vectors that can't be compared
		}

		// If we haven't found K vectors yet, just add to heap
		if h.Len() < k {
			heap.Push(h, SearchResult{Vector: vector, Similarity: similarity})
			continue
		}

		// If this vector is more similar than the least similar in our heap
		if similarity > (*h)[0].Similarity {
			heap.Pop(h)
			heap.Push(h, SearchResult{Vector: vector, Similarity: similarity})
		}
	}

	// Convert heap to sorted slice (most similar first)
	results := make([]SearchResult, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(h).(SearchResult)
	}

	return results, nil
}
