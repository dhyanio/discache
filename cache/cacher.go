package cache

import "time"

// Cacher is an interface for a cache
type Cacher interface {
	Put([]byte, []byte, time.Duration) error
	Has([]byte) bool
	Get([]byte) ([]byte, error)
}
