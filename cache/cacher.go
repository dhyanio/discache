package cache

import "time"

// Cacher represents a cache interface
type Cacher interface {
	Put([]byte, []byte, time.Duration) error
	Has([]byte) bool
	Get([]byte) ([]byte, error)
}
