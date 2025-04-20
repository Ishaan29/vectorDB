package storage

type Vector struct {
	ID        string
	Embedding []float32
	Metadata  map[string]interface{}
}
