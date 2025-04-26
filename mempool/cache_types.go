package mempool

import (
	"container/list"
	"sync"
	"time"
)

// EvictionPolicy determines how blocks are evicted from cache
type EvictionPolicy int

const (
	LRUEviction       EvictionPolicy = iota // Least Recently Used
	SizeBasedEviction                       // Evict based on block size
	HybridEviction                          // Combination of LRU and size-based
)

// EvictionConfig holds configuration for cache eviction
type EvictionConfig struct {
	Policy        EvictionPolicy
	MaxBlockSize  uint64  // Maximum block size to cache
	MinBlockSize  uint64  // Minimum block size to cache
	SizeThreshold float64 // Threshold for size-based eviction (0-1)
}

// CacheMetrics tracks statistics for the LRU cache
type CacheMetrics struct {
	Hits             uint64    // Number of successful cache hits
	Misses           uint64    // Number of cache misses
	Evictions        uint64    // Number of items evicted from cache
	TotalRequests    uint64    // Total number of cache requests
	HitRate          float64   // Cache hit rate (hits/total requests)
	LastMetricsReset time.Time // Last time metrics were reset
}

// WarmupStats tracks block allocation patterns
type WarmupStats struct {
	frequency map[uint64]uint64 // Maps block size to allocation count
	lastReset time.Time
}

// LRUNode represents a node in the LRU cache
type LRUNode struct {
	key   string    // Key to identify the block
	block *MemBlock // Pointer to the memory block
}

// LRUCache implements Least Recently Used cache
type LRUCache struct {
	capacity        uint64
	items           map[string]*list.Element
	lru             *list.List
	mu              sync.RWMutex
	metrics         *CacheMetrics
	evictionConfig  EvictionConfig
	totalCachedSize uint64 // Total size of all cached blocks
}
