package index

type VectorIndex interface {
	Add(id string, embedding []float32) error
}

type SearchResult struct {
	ID       string
	Distance float64
	Score    float64
}
