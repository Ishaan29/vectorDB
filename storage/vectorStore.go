package storage

import "github.com/ishaan29/vectorDB/pkg/types"

type VectorStore interface {
	Put(vector types.Vector) error
	Get(id string) (types.Vector, error)
	Delete(id string) error
	Search(query types.Vector, options types.SearchOptions) ([]types.SearchResult, error)
	GetIndexKey(id string) string
	GetMetadataKey(id string) string
	GetVectorKey(id string) string
	GetAllVectors() ([]types.Vector, error)
	Close() error
}
