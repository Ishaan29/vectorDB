package types

// In mem representation of a vector database
type Vector struct {
	ID        string                 `json:"id"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type SearchResult struct {
	Vector   Vector  `json:"vector"`
	Distance float32 `json:"distance"`
	Score    float32 `json:"score"`
}

type SearchOptions struct {
	K           int     `json:"k"`            // Number of results to return
	Threshold   float32 `json:"threshold"`    // Distance threshold
	IncludeVecs bool    `json:"include_vecs"` // Include vectors in results
	IncludeMeta bool    `json:"include_meta"` // Include metadata in results
}
