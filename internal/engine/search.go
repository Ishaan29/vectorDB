package engine

import (
	"github.com/ishaan29/vectorDB/pkg/types"
)

type SearchParams struct {
	K           int     // Number of results to return
	Threshold   float32 // Distance threshold
	IncludeVecs bool    // Include vectors in results
	IncludeMeta bool    // Include metadata in results
}

type resultHeap []types.SearchResult

func (h resultHeap) Len() int            { return len(h) }
func (h resultHeap) Less(i, j int) bool  { return h[i].Score < h[j].Score }
func (h resultHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *resultHeap) Push(x interface{}) { *h = append(*h, x.(types.SearchResult)) }
func (h *resultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Search performs a k-nearest neighbor search
// func (e *Engine) Search(query types.Vector, params SearchParams) ([]types.SearchResult, error) {
// 	e.mu.RLock()
// 	defer e.mu.RUnlock()

// 	if !e.running {
// 		return nil, ErrEngineNotRunning
// 	}

// 	if len(query.Embedding) != e.config.Index.Dimensions {
// 		return nil, fmt.Errorf("invalid dimensions: expected %d, got %d",
// 			e.config.Index.Dimensions, len(query.Embedding))
// 	}

// 	// Initialize priority queue for top-k results
// 	pq := make(resultHeap, 0, params.K)
// 	heap.Init(&pq)

// 	foundAny := false
// 	// Perform brute-force search
// 	for _, vec := range e.store {
// 		similarity, err := vectormath.CosineSimilarity(query.Embedding, vec.Embedding)
// 		if err != nil {
// 			e.logger.Warn("Failed to calculate similarity",
// 				logger.Error("error", err),
// 				logger.String("vector_id", vec.ID))
// 			continue
// 		}

// 		if similarity < params.Threshold {
// 			continue
// 		}

// 		foundAny = true
// 		result := types.SearchResult{
// 			Vector:   vec,
// 			Distance: float32(1 - similarity),
// 			Score:    float32(similarity),
// 		}

// 		if !params.IncludeVecs {
// 			// Only include ID if vectors are not requested
// 			result.Vector = types.Vector{
// 				ID: vec.ID,
// 			}
// 		}

// 		if !params.IncludeMeta {
// 			result.Vector.Metadata = nil
// 		}

// 		if pq.Len() < params.K {
// 			heap.Push(&pq, result)
// 		} else if pq[0].Score < similarity {
// 			heap.Pop(&pq)
// 			heap.Push(&pq, result)
// 		}
// 	}

// 	if !foundAny {
// 		return []types.SearchResult{}, nil
// 	}

// 	// Convert heap to sorted slice
// 	results := make([]types.SearchResult, pq.Len())
// 	for i := len(results) - 1; i >= 0; i-- {
// 		results[i] = heap.Pop(&pq).(types.SearchResult)
// 	}

// 	return results, nil
// }
