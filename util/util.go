package util

import (
	"crypto/rand"
	"fmt"
	"io"
)

// ExpiredKeyError is an error type for expired keys
type ExpiredKeyError struct {
	Key string
}

func (e *ExpiredKeyError) Error() string {
	return fmt.Sprintf("expired key %s", e.Key)
}

// KeyNotFoundError is an error type for missing keys
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %s not found", e.Key)
}

// randomByte return random bytes
func randomByte(n int) []byte {
	buf := make([]byte, n)
	io.ReadFull(rand.Reader, buf)
	return buf
}
