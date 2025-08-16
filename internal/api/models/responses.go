package models

import "github.com/ishaan29/vectorDB/pkg/types"

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type VectorResponse struct {
	ID        string                 `json:"id"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type SearchResult struct {
	ID       string          `json:"id"`
	Score    float32         `json:"score"`
	Distance float32         `json:"distance"`
	Vector   *VectorResponse `json:"vector,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	TookMs  int64          `json:"took_ms"`
	Total   int            `json:"total"`
}

type BatchInsertResponse struct {
	Success       bool  `json:"success"`
	Inserted      int   `json:"inserted"`
	Failed        int   `json:"failed"`
	TookMs        int64 `json:"took_ms"`
	FailedVectors []struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	} `json:"failed_vectors,omitempty"`
}

type HealthResponse struct {
	Status  string                 `json:"status"`
	Engine  string                 `json:"engine"`
	Version string                 `json:"version,omitempty"`
	Stats   map[string]interface{} `json:"stats,omitempty"`
}

type StatsResponse struct {
	Stats  map[string]interface{} `json:"stats"`
	TookMs int64                  `json:"took_ms"`
}

func ConvertVector(v types.Vector, includeEmbedding, includeMetadata bool) VectorResponse {
	resp := VectorResponse{
		ID: v.ID,
	}

	if includeEmbedding {
		resp.Embedding = v.Embedding
	}

	if includeMetadata {
		resp.Metadata = v.Metadata
	}

	return resp
}

func ConvertSearchResult(sr types.SearchResult, includeVectors, includeMetadata bool) SearchResult {
	result := SearchResult{
		ID:       sr.Vector.ID,
		Score:    sr.Score,
		Distance: sr.Distance,
	}

	if includeVectors || includeMetadata {
		vectorResp := ConvertVector(sr.Vector, includeVectors, includeMetadata)
		result.Vector = &vectorResp
	}

	return result
}
