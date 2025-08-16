package engine

import (
	"errors"
	"fmt"
)

var (
	ErrEngineNotRunning     = errors.New("engine is not running")
	ErrStoreInitialization  = errors.New("failed to initialize vector store")
	ErrIndexInitialization  = errors.New("failed to initialize index")
	ErrEngineAlreadyRunning = errors.New("engine is already running")
	ErrSearchIndexFailed    = errors.New("failed to search index")
	ErrVectorNotFound       = errors.New("vector not found")
)

func ErrInvalidDimensions(expected, actual int) error {
	return fmt.Errorf("invalid dimensions: expected %d, got %d",
		expected, actual)
}
