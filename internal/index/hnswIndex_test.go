package index

import (
	"testing"
)

func TestHNSWIndex_Add(t *testing.T) {
	idx := NewHNSWIndex(3, nil)

	// Add test vectors
	idx.Add("v1", []float32{1, 0, 0})
	idx.Add("v2", []float32{0, 1, 0})
	idx.Add("v3", []float32{0, 0, 1})

	// Search
	results, err := idx.Search([]float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].ID != "v1" {
		t.Errorf("Expected v1 as first result, got %s", results[0].ID)
	}
}
