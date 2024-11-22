package cache

import (
	"sync"
	"time"

	"github.com/dhyanio/discache/util"
)

// CacheOpts contains the configuration options for a cache
type CacheOpts struct {
	Capacity int
	TTL      time.Duration
	OnEvict  func(key string, value []byte)
}

// Cache is an in-memory key-value store with a fixed capacity and TTL
type Cache struct {
	CacheOpts
	items                   map[string][]byte
	order                   []string // Slice to maintain the LRU order
	mu                      sync.RWMutex
	hits, misses, evictions int
	timestamps              map[string]time.Time
}

// NewCache creates a new cache with the specified capacity, TTL, and eviction callback
func NewCache(opts CacheOpts) *Cache {
	return &Cache{
		CacheOpts:  opts,
		items:      make(map[string][]byte),
		order:      []string{},
		timestamps: make(map[string]time.Time),
	}
}

// Get retrieves an item from the cache and updates its usage
func (c *Cache) Get(key []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	strKey := string(key)

	if value, found := c.items[strKey]; found {
		if c.CacheOpts.TTL > 0 && time.Since(c.timestamps[strKey]) > c.CacheOpts.TTL {
			c.mu.RUnlock()
			c.mu.Lock()
			c.remove(strKey) // Expire the item if TTL has elapsed
			c.mu.Unlock()
			c.mu.RLock()
			c.misses++
			return nil, &util.ExpiredKeyError{Key: strKey}
		}
		c.hits++
		c.updateOrder(strKey) // Move the accessed key to the end of the order slice
		return value, nil
	}
	c.misses++
	return nil, &util.KeyNotFoundError{Key: strKey}
}

// Put inserts an item into the cache and updates its usage
func (c *Cache) Put(key, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	strKey := string(key)

	if _, found := c.items[strKey]; found {
		c.items[strKey] = value
		c.timestamps[strKey] = time.Now()
		c.updateOrder(strKey)
		return nil
	}

	// Evict the least recently used item if capacity is reached
	if len(c.items) >= c.CacheOpts.Capacity {
		c.evict()
	}

	c.items[strKey] = value
	c.timestamps[strKey] = time.Now()
	c.order = append(c.order, strKey) // Add key to the end of order slice
	return nil
}

// Has checks if a key exists in the cache
func (c *Cache) Has(key []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	strKey := string(key)
	if _, found := c.items[strKey]; found {
		if c.CacheOpts.TTL > 0 && time.Since(c.timestamps[strKey]) > c.CacheOpts.TTL {
			return false
		}
		return true
	}
	return false
}

// Stats returns the cache hit, miss, and eviction counts
func (c *Cache) Stats() (hits, misses, evictions int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.evictions
}

// evict removes the least recently used item from the cache
func (c *Cache) evict() {
	if len(c.order) == 0 {
		return
	}
	oldestKey := c.order[0]
	c.remove(oldestKey)
	c.evictions++
}

// remove deletes an item from the cache
func (c *Cache) remove(key string) {
	if value, found := c.items[key]; found {
		delete(c.items, key)
		delete(c.timestamps, key)
		if c.CacheOpts.OnEvict != nil {
			c.CacheOpts.OnEvict(key, value)
		}
		// Remove the key from the order slice
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
	}
}

// updateOrder moves a key to the end of the LRU order slice
func (c *Cache) updateOrder(key string) {
	for i, k := range c.order {
		if k == key {
			// Remove the key from its current position
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	// Add the key to the end to mark it as recently used
	c.order = append(c.order, key)
}
