package vectormath

import (
	"math"

	"github.com/ishaan29/vectorDB/internal/logger"
)

// cos(angle) = (A . B) / (||A|| * ||B||)
func CosineSimilarity(a []float32, b []float32) (float32, error) {
	if len(a) != len(b) {
		logger.Error("Dimension mismatch in CosineSimilarity",
			ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}
	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		logger.Error("Zero vector in CosineSimilarity",
			ErrZeroVector)
		return 0, ErrZeroVector
	}
	return dotProduct / float32(math.Sqrt(float64(normA)*float64(normB))), nil
}

func EuclideanDistance(a []float32, b []float32) (float32, error) {
	if len(a) != len(b) {
		logger.Error("Dimension mismatch in EuclideanDistance",
			ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += (a[i] - b[i]) * (a[i] - b[i])
	}
	return float32(math.Sqrt(float64(sum))), nil
}

func DotProduct(a []float32, b []float32) (float32, error) {
	if len(a) != len(b) {
		logger.Error("Dimension mismatch in DotProduct",
			ErrDimensionMismatch)
		return 0, ErrDimensionMismatch
	}
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum, nil
}
