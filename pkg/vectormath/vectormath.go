package vectormath

import (
	"math"

	"github.com/ishaan29/vectorDB/internal/logger"
)

// Dot computes the dot product of two vectors
func Dot(v1, v2 []float32) (float64, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in Dot", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}

	sum := 0.0
	for i := 0; i < len(v1); i++ {
		sum += float64(v1[i] * v2[i])
	}
	return sum, nil
}

// Magnitude returns the L2 norm (Euclidean norm) of the vector
func Magnitude(v []float32) float64 {
	sum := 0.0
	for _, val := range v {
		sum += float64(val * val)
	}
	return math.Sqrt(sum)
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(v1, v2 []float32) (float64, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in CosineSimilarity", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}

	dot, err := Dot(v1, v2)
	if err != nil {
		return 0, err
	}

	magV1 := Magnitude(v1)
	magV2 := Magnitude(v2)

	if magV1 == 0 || magV2 == 0 {
		logger.Error("Zero vector in CosineSimilarity", ErrZeroVector)
		return 0, ErrZeroVector
	}

	return dot / (magV1 * magV2), nil
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(v1, v2 []float32) (float64, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in EuclideanDistance", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}

	sum := 0.0
	for i := 0; i < len(v1); i++ {
		diff := float64(v1[i] - v2[i])
		sum += diff * diff
	}
	return math.Sqrt(sum), nil
}

// NormalizeVector normalizes a vector in place to have unit magnitude
func NormalizeVector(v []float32) error {
	mag := Magnitude(v)
	if mag == 0 {
		logger.Error("Cannot normalize zero vector", ErrZeroVector)
		return ErrZeroVector
	}

	for i := range v {
		v[i] = float32(float64(v[i]) / mag)
	}
	return nil
}
