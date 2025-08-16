package index

import (
	"fmt"
	"sync"

	"github.com/fogfish/hnsw"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
	"github.com/ishaan29/vectorDB/pkg/vectormath"
)

// CosineSurface implements vector.Surface for cosine distance
type CosineSurface struct{}

// Distance calculates cosine distance between two vectors
func (CosineSurface) Distance(a, b types.Vector) float32 {
	dist, err := vectormath.CosineDistance(a.Embedding, b.Embedding)
	if err != nil {
		// Return maximum distance on error
		return 2.0
	}
	return dist
}

// Equal checks if two vectors are equal by ID
func (CosineSurface) Equal(a, b types.Vector) bool {
	return a.ID == b.ID
}

// HNSWIndex wraps the fogfish HNSW implementation
type HNSWIndex struct {
	mu       sync.RWMutex
	index    *hnsw.HNSW[types.Vector]
	dim      int
	logger   logger.Logger
	vectors  map[string]types.Vector // Track vectors by ID
	efSearch int
}

func NewHNSWIndex(dimensions int, log logger.Logger) *HNSWIndex {
	m := 16               // Number of bi-directional links
	efConstruction := 200 // Size of the dynamic candidate list

	// Create the distance function (Surface)
	surface := CosineSurface{}

	// Create HNSW index
	index := hnsw.New[types.Vector](
		surface,
		hnsw.WithM(m),
		hnsw.WithEfConstruction(efConstruction),
	)

	return &HNSWIndex{
		index:    index,
		dim:      dimensions,
		logger:   log,
		vectors:  make(map[string]types.Vector),
		efSearch: 50, // Default search effort
	}
}

func (h *HNSWIndex) Add(id string, embedding []float32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate dimensions
	if len(embedding) != h.dim {
		return fmt.Errorf("dimension mismatch: expected %d, got %d",
			h.dim, len(embedding))
	}

	// Check if already exists
	if _, exists := h.vectors[id]; exists {
		if h.logger != nil {
			h.logger.Debug("Vector already in index, skipping",
				logger.String("id", id))
		}
		return nil
	}

	// Create vector
	vector := types.Vector{
		ID:        id,
		Embedding: embedding,
		// Metadata is not stored in index
	}

	// Insert into HNSW (not Add!)
	h.index.Insert(vector)

	// Track vector
	h.vectors[id] = vector

	if h.logger != nil {
		h.logger.Debug("Inserted vector into HNSW index",
			logger.String("id", id),
			logger.Int("total_vectors", len(h.vectors)))
	}

	return nil
}

func (h *HNSWIndex) Search(query []float32, k int) ([]SearchResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Validate dimensions
	if len(query) != h.dim {
		return nil, fmt.Errorf("query dimension mismatch: expected %d, got %d",
			h.dim, len(query))
	}

	// Handle empty index
	if len(h.vectors) == 0 {
		return []SearchResult{}, nil
	}

	// Adjust k if we have fewer vectors
	if k > len(h.vectors) {
		k = len(h.vectors)
	}

	// Create query vector
	queryVector := types.Vector{
		ID:        "", // Anonymous query
		Embedding: query,
	}

	// Search returns []Vector directly
	neighbors := h.index.Search(queryVector, k, h.efSearch)

	// Convert to our result format
	results := make([]SearchResult, 0, len(neighbors))
	for _, vec := range neighbors {
		// Calculate distance and similarity
		distance, _ := vectormath.CosineDistance(query, vec.Embedding)
		similarity, _ := vectormath.CosineSimilarity(query, vec.Embedding)

		results = append(results, SearchResult{
			ID:       vec.ID,
			Distance: float64(distance),
			Score:    float64(similarity),
		})
	}

	if h.logger != nil {
		h.logger.Debug("Search completed",
			logger.Int("results", len(results)),
			logger.Int("requested_k", k))
	}

	return results, nil
}

func (h *HNSWIndex) Remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// HNSW doesn't support removal, just remove from tracking
	if _, exists := h.vectors[id]; exists {
		delete(h.vectors, id)
		if h.logger != nil {
			h.logger.Warn("Removed from tracking (node remains in HNSW graph)",
				logger.String("id", id))
		}
		return nil
	}

	return fmt.Errorf("vector %s not found in index", id)
}

func (h *HNSWIndex) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.index.Size() // Use the actual index size
}

func (h *HNSWIndex) SetSearchEf(ef int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ef < 1 {
		ef = 1
	}
	if ef > 1000 {
		ef = 1000
	}

	h.efSearch = ef

	if h.logger != nil {
		h.logger.Info("Updated search effort",
			logger.Int("efSearch", ef))
	}
}

func (h *HNSWIndex) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"vectors":    h.index.Size(),
		"dimensions": h.dim,
		"ef_search":  h.efSearch,
		"levels":     h.index.Level(),
	}
}
