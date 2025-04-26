package types

import (
	"math"
)

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

// MathVector represents a mathematical vector of float64 values
type MathVector struct {
	Values []float64
}

// NewMathVector creates a new vector from a slice of float64 values
func NewMathVector(values []float64) *MathVector {
	v := make([]float64, len(values))
	copy(v, values)
	return &MathVector{Values: v}
}

// Dot computes the dot product with another vector
func (v *MathVector) Dot(other *MathVector) float64 {
	if len(v.Values) != len(other.Values) {
		return 0.0
	}

	sum := 0.0
	for i := 0; i < len(v.Values); i++ {
		sum += v.Values[i] * other.Values[i]
	}
	return sum
}

// Magnitude returns the L2 norm (Euclidean norm) of the vector
func (v *MathVector) Magnitude() float64 {
	sum := 0.0
	for _, val := range v.Values {
		sum += val * val
	}
	return math.Sqrt(sum)
}

// CosineSimilarity calculates the cosine similarity with another vector
func (v *MathVector) CosineSimilarity(other *MathVector) float64 {
	if len(v.Values) != len(other.Values) {
		return 0.0
	}

	dot := v.Dot(other)
	magV := v.Magnitude()
	magOther := other.Magnitude()

	if magV == 0 || magOther == 0 {
		return 0.0
	}

	return dot / (magV * magOther)
}
