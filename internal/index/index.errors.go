package index

import "fmt"

func ErrDimensionMismatch(dim int, embeddingLen int) error {
	return fmt.Errorf("dimension mismatch: expected %d, got %d", dim, embeddingLen)
}
