package mempool

import (
	"container/list"
	"time"
)

func NewLRUCache(capacity uint64) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
		metrics: &CacheMetrics{
			LastMetricsReset: time.Now(),
		},
	}
}

func (c *LRUCache) Get(key string) (*MemBlock, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalRequests++
	if element, exists := c.items[key]; exists {
		c.lru.MoveToFront(element)
		c.metrics.Hits++
		c.updateHitRate()
		return element.Value.(*LRUNode).block, true
	}
	c.metrics.Misses++
	c.updateHitRate()
	return nil, false
}

func (c *LRUCache) Put(key string, block *MemBlock) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if block size is within acceptable range
	if block.Header.Size < c.evictionConfig.MinBlockSize ||
		block.Header.Size > c.evictionConfig.MaxBlockSize {
		return // Don't cache blocks outside size range
	}

	if element, exists := c.items[key]; exists {
		c.lru.MoveToFront(element)
		oldBlock := element.Value.(*LRUNode).block
		c.totalCachedSize -= oldBlock.Header.Size
		element.Value.(*LRUNode).block = block
		c.totalCachedSize += block.Header.Size
		return
	}

	// Check if adding this block would exceed capacity
	for c.totalCachedSize+block.Header.Size > c.capacity && c.lru.Len() > 0 {
		c.Evict()
	}

	node := &LRUNode{
		key:   key,
		block: block,
	}
	element := c.lru.PushFront(node)
	c.items[key] = element
	c.totalCachedSize += block.Header.Size
}

func (c *LRUCache) Remove(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.lru.Remove(element)
		delete(c.items, key)
		return true
	}
	return false
}

func (c *LRUCache) Evict() {
	if element := c.lru.Back(); element != nil {
		node := element.Value.(*LRUNode)
		block := node.block

		if c.shouldEvictBlock(block) {
			delete(c.items, node.key)
			c.lru.Remove(element)
			c.totalCachedSize -= block.Header.Size
			c.metrics.Evictions++
		}
	}
}

func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lru.Len()
}

func (c *LRUCache) updateHitRate() {
	c.metrics.HitRate = float64(c.metrics.Hits) / float64(c.metrics.TotalRequests)
}

// GetMetrics returns a copy of current cache metrics
func (c *LRUCache) GetMetrics() CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c.metrics
}

// ResetMetrics resets all cache metrics
func (c *LRUCache) ResetMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = &CacheMetrics{
		LastMetricsReset: time.Now(),
	}
}

// shouldEvictBlock determines if a block should be evicted based on policy
func (c *LRUCache) shouldEvictBlock(block *MemBlock) bool {
	switch c.evictionConfig.Policy {
	case SizeBasedEviction:
		return block.Header.Size > c.evictionConfig.MaxBlockSize ||
			block.Header.Size < c.evictionConfig.MinBlockSize

	case HybridEviction:
		// Consider both size and LRU
		isSize := block.Header.Size > c.evictionConfig.MaxBlockSize ||
			block.Header.Size < c.evictionConfig.MinBlockSize
		isLRU := c.lru.Back().Value.(*LRUNode).block == block
		return isSize || isLRU

	default: // LRUEviction
		return true
	}
}
