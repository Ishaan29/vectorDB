package persistence

import (
	"errors"
	"fmt"
)

var (
	ErrBadgerOpen    = errors.New("failed to open badger db")
	ErrBadgerPut     = errors.New("failed to put vector")
	ErrBadgerMarshal = errors.New("failed to marshal data")
	ErrBadgerGet     = errors.New("failed to get vector")
	ErrBadgerDelete  = errors.New("failed to delete vector")
	ErrBadgerClose   = errors.New("failed to close badger db")
)

func ErrBadgerKeyNotFound(id string) error {
	return fmt.Errorf("vector not found %s", id)
}

func ErrBadgerBatchMarshal(id string, err error) error {
	return fmt.Errorf("failed to marshal vector %s: %w", id, err)
}

func ErrBadgerBatchSet(id string, err error) error {
	return fmt.Errorf("failed to set vector %s: %w", id, err)
}

func ErrBadgerBatchWriteFailed(index int, err error) error {
	return fmt.Errorf("batch write failed at index %d: %w", index, err)
}
