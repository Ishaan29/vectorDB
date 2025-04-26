package mempool

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// CacheWarmingStrategy defines how to warm up the cache
type CacheWarmingStrategy int

const (
	NoWarming      CacheWarmingStrategy = iota
	FrequencyBased                      // Warm based on allocation frequency
	SizeBased                           // Warm based on block sizes
	HybridWarming                       // Combination of frequency and size
)

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Block      *MemBlock
	Similarity float64
}

// Error definitions
var (
	ErrInvalidK        = fmt.Errorf("k must be greater than 0")
	ErrInvalidDataType = fmt.Errorf("invalid data type: expected []byte")
)

// Block represents a data block in memory
type Block struct {
	Header BlockHeader
	Data   interface{}
}

func NewMemPool(config PoolConfig) (*MemPool, error) {
	if config.InitialSize == 0 {
		return nil, ErrInvalidSize
	}
	if config.MinBlockSize == 0 || config.MaxBlockSize == 0 {
		return nil, ErrInvalidSize
	}

	if config.MinBlockSize > config.MaxBlockSize {
		return nil, ErrInvalidSize
	}

	pool := &MemPool{
		block: make([]*MemBlock, 0),
		metrics: MemMatrics{
			TotalSize:        config.InitialSize,
			AvailableSize:    config.InitialSize,
			LastMetricsReset: time.Now(),
		},
		minBlockSize: config.MinBlockSize,
		maxBlockSize: config.MaxBlockSize,
		cacheSize:    config.CacheSize,
	}

	// Initialize LRU cache if cache size > 0
	if config.CacheSize > 0 {
		pool.cache = NewLRUCache(config.CacheSize)
	}

	intialBlock := &MemBlock{
		Header: BlockHeader{
			Size:         config.InitialSize,
			IsAllocated:  false,
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
		},
		Data: make([]byte, config.InitialSize),
	}

	pool.block = append(pool.block, intialBlock)
	return pool, nil
}

func (p *MemPool) GetMatrics() MemMatrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

func (p *MemPool) ResetMetrics() {
	p.mu.Lock()

	p.metrics.Allocations = 0
	p.metrics.Deallocations = 0
	p.metrics.LastMetricsReset = time.Now()

	defer p.mu.Unlock()
}

func (p *MemPool) UpdateMetrics() {
	var used uint64
	for _, block := range p.block {
		if block.Header.IsAllocated {
			used += block.Header.Size
		}
	}

	p.metrics.UsedSize = used
	p.metrics.AvailableSize = p.metrics.TotalSize - used
}

// tryGetFromCache attempts to retrieve a block from cache
func (p *MemPool) tryGetFromCache(size uint64) (*MemBlock, bool) {
	if p.cache == nil {
		return nil, false
	}

	// Try to find a cached block of appropriate size
	key := fmt.Sprintf("size_%d", size)
	if block, found := p.cache.Get(key); found {
		return block, true
	}
	return nil, false
}

// addToCache adds a block to the cache
func (p *MemPool) addToCache(block *MemBlock) {
	if p.cache == nil {
		return
	}
	key := fmt.Sprintf("size_%d", block.Header.Size)
	p.cache.Put(key, block)
}

// Modified Alloc to use cache
func (p *MemPool) Alloc(size uint64) (*MemBlock, error) {
	if size == 0 || size > p.maxBlockSize {
		return nil, ErrInvalidSize
	}

	// Track allocation for cache warming
	p.TrackAllocation(size)

	// Try cache first
	if block, found := p.tryGetFromCache(size); found {
		return block, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Original allocation logic
	i := 0
	for i < len(p.block) {
		block := p.block[i]
		if !block.Header.IsAllocated && block.Header.Size >= size {
			if block.Header.Size <= size+p.minBlockSize {
				block.Header.IsAllocated = true
				block.Header.LastAccessed = time.Now()
				p.metrics.Allocations++
				p.UpdateMetrics()
				return block, nil
			}

			// split
			remainingSize := block.Header.Size - size
			block.Header.Size = size
			block.Header.IsAllocated = true
			block.Header.LastAccessed = time.Now()
			if data, ok := block.Data.([]byte); ok {
				block.Data = data[:size]
			}

			newBlock := &MemBlock{
				Header: BlockHeader{
					Size:         remainingSize,
					IsAllocated:  false,
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
				},
				Data: make([]byte, remainingSize),
			}

			p.block = append(p.block[:i+1], append([]*MemBlock{newBlock}, p.block[i+1:]...)...)
			p.metrics.Allocations++
			p.UpdateMetrics()

			// Add the new free block to cache
			p.addToCache(newBlock)
			return block, nil
		}
		i++
	}
	return nil, ErrAllocationFaild
}

// Free marks a block as deallocated and attempts to merge with adjacent free blocks
func (p *MemPool) Free(block *MemBlock) error {
	if block == nil {
		return ErrInvalidBlock
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	blockIndex := -1
	for i, b := range p.block {
		if b == block {
			blockIndex = i
			break
		}
	}

	if blockIndex == -1 {
		return ErrBlockNotAllocated
	}

	// mark block as free
	block.Header.IsAllocated = false
	block.Header.LastAccessed = time.Now()

	// Try to merge with previous block
	if blockIndex > 0 && !p.block[blockIndex-1].Header.IsAllocated {
		prevBlock := p.block[blockIndex-1]
		prevBlock.Header.Size += block.Header.Size

		// Create new merged data slice
		newData := make([]byte, prevBlock.Header.Size)
		if prevData, ok := prevBlock.Data.([]byte); ok {
			copy(newData, prevData)
		}
		if blockData, ok := block.Data.([]byte); ok {
			copy(newData[len(newData)-len(blockData):], blockData)
		}
		prevBlock.Data = newData

		// Remove the current block
		p.block = append(p.block[:blockIndex], p.block[blockIndex+1:]...)
		blockIndex--
		block = prevBlock
	}

	// Try to merge with next block
	if blockIndex < len(p.block)-1 && !p.block[blockIndex+1].Header.IsAllocated {
		nextBlock := p.block[blockIndex+1]
		block.Header.Size += nextBlock.Header.Size

		// Create new merged data slice
		newData := make([]byte, block.Header.Size)
		if blockData, ok := block.Data.([]byte); ok {
			copy(newData, blockData)
		}
		if nextData, ok := nextBlock.Data.([]byte); ok {
			copy(newData[len(newData)-len(nextData):], nextData)
		}
		block.Data = newData

		// Remove the next block
		p.block = append(p.block[:blockIndex+1], p.block[blockIndex+2:]...)
	}

	p.metrics.Deallocations++
	p.UpdateMetrics()

	// Add to cache for potential reuse
	p.addToCache(block)
	return nil
}

// Defrag consolidates free blocks to reduce fragmentation
func (p *MemPool) Defrag() {
	p.mu.Lock()
	defer p.mu.Unlock()

	i := 0
	for i < len(p.block)-1 {
		current := p.block[i]
		next := p.block[i+1]

		if !current.Header.IsAllocated && !next.Header.IsAllocated {
			// Merge blocks
			current.Header.Size += next.Header.Size

			// Create new merged data slice
			newData := make([]byte, current.Header.Size)
			if currentData, ok := current.Data.([]byte); ok {
				copy(newData, currentData)
			}
			if nextData, ok := next.Data.([]byte); ok {
				copy(newData[len(newData)-len(nextData):], nextData)
			}
			current.Data = newData

			// Remove the next block
			p.block = append(p.block[:i+1], p.block[i+2:]...)
			continue
		}
		i++
	}
	p.UpdateMetrics()
}

func (p *MemPool) GetFragmentationInfo() (blocks int, freeBlocks int, largestFreeBlock uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	blocks = len(p.block)
	for _, block := range p.block {
		if !block.Header.IsAllocated {
			freeBlocks++
			if block.Header.Size > largestFreeBlock {
				largestFreeBlock = block.Header.Size
			}
		}
	}
	return
}

// GetCacheMetrics returns current cache statistics
func (p *MemPool) GetCacheMetrics() *CacheMetrics {
	if p.cache == nil {
		return nil
	}
	metrics := p.cache.GetMetrics()
	return &metrics
}

// ResetCacheMetrics resets cache statistics
func (p *MemPool) ResetCacheMetrics() {
	if p.cache != nil {
		p.cache.ResetMetrics()
	}
}

// GetCacheEfficiency returns the current cache hit rate as a percentage
func (p *MemPool) GetCacheEfficiency() float64 {
	if p.cache == nil {
		return 0.0
	}
	metrics := p.cache.GetMetrics()
	return metrics.HitRate * 100
}

// WarmCache pre-populates the cache based on usage patterns
func (p *MemPool) WarmCache(strategy CacheWarmingStrategy) {
	if p.cache == nil {
		return
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	switch strategy {
	case FrequencyBased:
		p.warmByFrequency()
	case SizeBased:
		p.warmBySize()
	case HybridWarming:
		p.warmHybrid()
	}
}

// warmByFrequency warms cache with most frequently allocated block sizes
func (p *MemPool) warmByFrequency() {
	// Find blocks with sizes matching frequent allocations
	for i := 0; i < len(p.block); i++ {
		block := p.block[i]
		if !block.Header.IsAllocated {
			p.addToCache(block)
		}
	}
}

// warmBySize warms cache with blocks in optimal size range
func (p *MemPool) warmBySize() {
	minSize := p.minBlockSize
	maxSize := p.maxBlockSize / 2 // Target medium-sized blocks

	for i := 0; i < len(p.block); i++ {
		block := p.block[i]
		if !block.Header.IsAllocated &&
			block.Header.Size >= minSize &&
			block.Header.Size <= maxSize {
			p.addToCache(block)
		}
	}
}

// warmHybrid combines frequency and size-based warming
func (p *MemPool) warmHybrid() {
	// First warm by frequency
	p.warmByFrequency()

	// Then add size-appropriate blocks if cache isn't full
	if p.cache.totalCachedSize < p.cache.capacity {
		p.warmBySize()
	}
}

// TrackAllocation records allocation patterns for warming
func (p *MemPool) TrackAllocation(size uint64) {
	if p.warmupStats.frequency == nil {
		p.warmupStats.frequency = make(map[uint64]uint64)
		p.warmupStats.lastReset = time.Now()
	}
	p.warmupStats.frequency[size]++
}

// Search finds the k most similar blocks to the query vector
func (p *MemPool) Search(query interface{}, k int) []Block {
	p.mu.RLock()
	defer p.mu.RUnlock()

	queryBytes, ok := query.([]byte)
	if !ok {
		return nil
	}

	type Result struct {
		block      Block
		similarity float64
	}

	var results []Result

	// Calculate similarities for all blocks
	for _, memBlock := range p.block {
		if !memBlock.Header.IsAllocated {
			continue
		}

		blockData, ok := memBlock.Data.([]byte)
		if !ok {
			continue
		}

		similarity := p.calculateSimilarity(queryBytes, blockData)
		block := Block{
			Header: memBlock.Header,
			Data:   memBlock.Data,
		}
		results = append(results, Result{block: block, similarity: similarity})
	}

	// Sort results by similarity in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// Return top k results
	k = min(k, len(results))
	topK := make([]Block, k)
	for i := 0; i < k; i++ {
		topK[i] = results[i].block
	}

	return topK
}

// calculateSimilarity computes the similarity between two byte slices
func (p *MemPool) calculateSimilarity(a, b []byte) float64 {
	// Simple cosine similarity implementation
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
