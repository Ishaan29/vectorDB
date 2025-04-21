package vectormath

import "errors"

var (
	// ErrDimensionMismatch is returned when two vectors have different dimensions
	ErrDimensionMismatch = errors.New("vectors have different dimensions")

	// ErrZeroVector is returned when a vector has zero magnitude
	ErrZeroVector = errors.New("vector has zero magnitude")
)
