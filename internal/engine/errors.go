package engine

import "errors"

var (
	// ErrEngineNotRunning is returned when trying to perform operations on a stopped engine
	ErrEngineNotRunning = errors.New("engine is not running")

	// ErrInvalidDimensions is returned when vector dimensions don't match the configured dimensions
	ErrInvalidDimensions = errors.New("invalid vector dimensions")
)
