// Package vectormath provides float32 vector math for the database.
//
// The public functions validate inputs and report errors; the arithmetic
// itself runs in a pluggable kernel selected at build time:
//   - CGO_ENABLED=1: C++ SIMD core (pkg/vectormath/simd)
//   - CGO_ENABLED=0: pure-Go kernels (pkg/vectormath/scalar)
//
// Use Backend() to inspect which implementation is active.
package vectormath

import (
	"math"

	"github.com/ishaan29/vectorDB/internal/logger"
)

// Dot computes the dot product of two vectors
func Dot(v1, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in Dot", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}
	return dotKernel(v1, v2), nil
}

// Magnitude returns the L2 norm (Euclidean norm) of the vector
func Magnitude(v []float32) float32 {
	return float32(math.Sqrt(float64(sqNormKernel(v))))
}

// CosineSimilarity calculates the cosine similarity between two vectors.
// The dot product and both norms are computed in a single fused pass.
func CosineSimilarity(v1, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in CosineSimilarity", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}

	sim, ok := cosineKernel(v1, v2)
	if !ok {
		logger.Error("Zero vector in CosineSimilarity", ErrZeroVector)
		return 0, ErrZeroVector
	}
	return sim, nil
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(v1, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		logger.Error("Dimension mismatch in EuclideanDistance", ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}
	return float32(math.Sqrt(float64(sqEuclideanKernel(v1, v2)))), nil
}

// NormalizeVector normalizes a vector in place to have unit magnitude
func NormalizeVector(v []float32) error {
	mag := Magnitude(v)
	if mag == 0 {
		logger.Error("Cannot normalize zero vector", ErrZeroVector)
		return ErrZeroVector
	}

	for i := range v {
		v[i] = v[i] / mag
	}
	return nil
}
