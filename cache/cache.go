package cache

import (
	"sync"
	"time"

	"github.com/dhyanio/discache/util"
)

// Cache represents LRU cache

type Cache struct {
	capacity                int
	ttl                     time.Duration
	items                   map[string][]byte
	order                   []string // Slice to maintain the LRU order
	mu                      sync.RWMutex
	hits, misses, evictions int
	onEvict                 func(key string, value []byte)
	timestamps              map[string]time.Time
}

// NewCache creates a new Cache
func NewCache(capacity int, ttl time.Duration, onEvict func(key string, value []byte)) *Cache {
	return &Cache{
		capacity:   capacity,
		ttl:        ttl,
		items:      make(map[string][]byte),
		order:      []string{},
		onEvict:    onEvict,
		timestamps: make(map[string]time.Time),
	}
}

// Get retrieves an item from the cache and updates its position in the LRU order.
func (c *Cache) Get(key []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	strKey := string(key)

	if value, found := c.items[strKey]; found {
		if c.ttl > 0 && time.Since(c.timestamps[string(key)]) > c.ttl {
			c.remove(strKey) // Expire the item if TTL has elapsed
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

// Put adds or updates an item in the cache
func (c *Cache) Put(key, value []byte, ttl time.Duration) error {
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
	if len(c.items) >= c.capacity {
		c.evict()
	}

	c.items[strKey] = value
	c.timestamps[strKey] = time.Now()
	c.order = append(c.order, strKey) // Add key to the end of order slice
	return nil
}

// Has checks if an item exists without updating its usage
func (c *Cache) Has(key []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	strKey := string(key)
	if _, found := c.items[strKey]; found {
		if c.ttl > 0 && time.Since(c.timestamps[strKey]) > c.ttl {
			return false
		}
		return true
	}
	return false
}

// Stats returns cache statistics
func (c *Cache) Stats() (hits, misses, evictions int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.evictions
}

// evict removes the least recently used item
func (c *Cache) evict() {
	if len(c.order) == 0 {
		return
	}
	oldestKey := c.order[0]
	c.remove(oldestKey)
	c.evictions++
}

// remove deletes a key from the cache and updates the LRU order
func (c *Cache) remove(key string) {
	if _, found := c.items[key]; found {
		delete(c.items, key)
		delete(c.timestamps, key)
		if c.onEvict != nil {
			c.onEvict(key, c.items[key])
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
