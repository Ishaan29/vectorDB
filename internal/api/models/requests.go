package models

type InsertRequest struct {
	ID        string                 `json:"id" binding:"required"`
	Embedding []float32              `json:"embedding" binding:"required"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type BatchInsertRequest struct {
	Vectors []InsertRequest `json:"vectors" binding:"required"`
}

type SearchRequest struct {
	Embedding       []float32 `json:"embedding" binding:"required"`
	K               int       `json:"k" binding:"required,min=1"`
	Threshold       float32   `json:"threshold,omitempty"`
	IncludeVectors  bool      `json:"include_vectors,omitempty"`
	IncludeMetadata bool      `json:"include_metadata,omitempty"`
}

type OptimizeRequest struct {
	Force bool `json:"force,omitempty"`
}
