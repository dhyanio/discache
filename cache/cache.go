package cache

import (
	"container/list"
	"sync"
	"time"
)

type Cache struct {
	capacity                int
	ttl                     time.Duration
	items                   map[int]*list.Element
	order                   *list.List
	mu                      sync.RWMutex
	hits, misses, evictions int
	onEvict                 func(key int, value string)
}

type Entry struct {
	key       int
	value     string
	timestamp time.Time
}

func NewCache(capacity int, ttl time.Duration, onEvict func(key int, value string)) *Cache {
	return &Cache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[int]*list.Element),
		order:    list.New(),
		onEvict:  onEvict,
	}
}

// Get retrieves an item and refreshes its position
func (c *Cache) Get(key int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if element, found := c.items[key]; found {
		entry := element.Value.(*Entry)
		if c.ttl > 0 && time.Since(entry.timestamp) > c.ttl {
			c.removeElement(element) // Expire item
			c.misses++
			return "", false
		}
		c.mu.RUnlock()
		c.mu.Lock()
		c.order.MoveToFront(element)
		c.mu.Unlock()
		c.mu.RLock()
		c.hits++
		return entry.value, true
	}
	c.misses++
	return "", false
}

// Put adds or updates an item
func (c *Cache) Put(key int, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, found := c.items[key]; found {
		element.Value.(*Entry).value = value
		element.Value.(*Entry).timestamp = time.Now()
		c.order.MoveToFront(element)
		return
	}

	if c.order.Len() >= c.capacity {
		c.evict()
	}

	entry := &Entry{key: key, value: value, timestamp: time.Now()}
	element := c.order.PushFront(entry)
	c.items[key] = element
}

// Has checks if an item exists without marking it as used
func (c *Cache) Has(key int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	element, found := c.items[key]
	if !found {
		return false
	}

	if c.ttl > 0 && time.Since(element.Value.(*Entry).timestamp) > c.ttl {
		return false
	}
	return true
}

// Stats returns cache statistics
func (c *Cache) Stats() (hits, misses, evictions int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.evictions
}

// evict removes the least recently used item
func (c *Cache) evict() {
	oldest := c.order.Back()
	if oldest != nil {
		entry := oldest.Value.(*Entry)
		c.removeElement(oldest)
		if c.onEvict != nil {
			c.onEvict(entry.key, entry.value)
		}
		c.evictions++
	}
}

// removeElement removes an element from the list and map
func (c *Cache) removeElement(element *list.Element) {
	c.order.Remove(element)
	entry := element.Value.(*Entry)
	delete(c.items, entry.key)
}
