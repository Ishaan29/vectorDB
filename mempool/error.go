package mempool

import "errors"

var (
	ErrInvalidSize       = errors.New("Invalid pool size")
	ErrAllocationFaild   = errors.New("Allocation failed")
	ErrInvalidBlock      = errors.New("Invalid block pointer")
	ErrBlockNotAllocated = errors.New("Block is not allocated")
)
