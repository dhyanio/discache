package cache

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Cacher represents a cache interface
type Cache struct {
	lock sync.RWMutex
	data map[string][]byte
}

// NewCache creates a new cache
func NewCache() *Cache {
	return &Cache{
		data: make(map[string][]byte),
	}
}

// Delete deletes a key from the cache
func (c *Cache) Delete(key []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.data, string(key))

	return nil
}

// Has checks if a key exists in the cache
func (c *Cache) Has(key []byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.data[string(key)]
	return ok
}

// Get gets a key from the cache
func (c *Cache) Get(key []byte) ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	log.Printf("key [%s] requested from cache\n", key)

	keyStr := string(key)
	val, ok := c.data[keyStr]
	if !ok {
		return nil, fmt.Errorf("key (%s) not found", keyStr)
	}

	return val, nil
}

func (c *Cache) Set(key, value []byte, ttl time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.data[string(key)] = value

	if ttl > 0 {
		go func() {
			<-time.After(ttl)
			delete(c.data, string(key))
		}()
	}

	return nil
}
