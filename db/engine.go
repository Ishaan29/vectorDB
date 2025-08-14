package db

import (
	"container/heap"
	"log"
	"sync"

	"github.com/ishaan29/vectorDB/pkg/types"
	"github.com/ishaan29/vectorDB/pkg/vectormath"
	"github.com/ishaan29/vectorDB/storage"
)

type Engine struct {
	mu          sync.RWMutex
	store       map[string]storage.Vector
	vectorStore storage.VectorStore
	// index
}

func NewEngine() *Engine {
	return &Engine{
		store:       make(map[string]storage.Vector),
		vectorStore: nil,
	}
}

func (engine *Engine) Insert(vector storage.Vector) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	// engine.store[vector.ID] = vector
	vector.ID = engine.vectorStore.GetVectorKey(vector.ID)
	engine.vectorStore.Put(types.Vector{ID: vector.ID, Embedding: vector.Embedding, Metadata: vector.Metadata})
}

func (engine *Engine) Get(id string) (storage.Vector, bool) {
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	vector, err := engine.vectorStore.Get(id)
	if err != nil {
		return storage.Vector{}, false
	}
	return storage.Vector{ID: vector.ID, Embedding: vector.Embedding, Metadata: vector.Metadata}, true
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
	vectors, err := engine.vectorStore.GetAllVectors()
	if err != nil {
		log.Printf("Failed to get all vectors: %v", err)
		return nil, err
	}
	for _, v := range vectors {
		similarity, err := vectormath.CosineSimilarity(query, v.Embedding)
		if err != nil {
			continue // Skip vectors that can't be compared
		}

		// If we haven't found K vectors yet, just add to heap
		if h.Len() < k {
			heap.Push(h, SearchResult{Vector: storage.Vector{ID: v.ID, Embedding: v.Embedding, Metadata: v.Metadata}, Similarity: similarity})
			continue
		}

		// If this vector is more similar than the least similar in our heap
		if similarity > (*h)[0].Similarity {
			heap.Pop(h)
			heap.Push(h, SearchResult{Vector: storage.Vector{ID: v.ID, Embedding: v.Embedding, Metadata: v.Metadata}, Similarity: similarity})
		}
	}

	// Convert heap to sorted slice (most similar first)
	results := make([]SearchResult, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(h).(SearchResult)
	}

	return results, nil
}
