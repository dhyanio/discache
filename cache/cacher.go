package cache

import "time"

// Cacher is an interface that defines methods for a cache system.
// It includes methods to put data into the cache, check for the existence
// of data, and retrieve data from the cache.
type Cacher interface {
	// Put stores the given value in the cache with the specified key and
	// expiration duration. It returns an error if the operation fails.
	Put(key []byte, value []byte, duration time.Duration) error

	// Has checks if the given key exists in the cache. It returns true if
	// the key is found, otherwise false.
	Has(key []byte) bool

	// Get retrieves the value associated with the given key from the cache.
	// It returns the value and an error if the operation fails or the key
	// does not exist.
	Get(key []byte) ([]byte, error)
}
