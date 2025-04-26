package mempool

import (
	"sync"
	"time"
)

// MemMatrics tracks memory pool usage statistics
type MemMatrics struct {
	TotalSize        uint64
	UsedSize         uint64
	AvailableSize    uint64
	Allocations      uint64
	Deallocations    uint64
	LastMetricsReset time.Time
}

// BlockHeader contains metadata about a memory block
type BlockHeader struct {
	Size         uint64    // Size of the block in bytes
	IsAllocated  bool      // Whether the block is currently allocated
	CreatedAt    time.Time // When the block was created
	LastAccessed time.Time // When the block was last accessed
}

// MemBlock represents a block of memory
type MemBlock struct {
	Header BlockHeader
	Data   interface{} // Can store any type of data
}

// PoolConfig contains configuration for the memory pool
type PoolConfig struct {
	InitialSize  uint64
	MinBlockSize uint64
	MaxBlockSize uint64
	CacheSize    uint64 // Number of blocks to cache
}

// MemPool manages memory allocation and deallocation
type MemPool struct {
	mu           sync.RWMutex
	block        []*MemBlock
	metrics      MemMatrics
	minBlockSize uint64
	maxBlockSize uint64

	// Cache-related fields
	cache       *LRUCache
	cacheSize   uint64
	cacheStats  CacheMetrics
	warmupStats WarmupStats
}
