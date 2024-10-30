package util

import "fmt"

type ExpiredKeyError struct {
	Key string
}

func (e *ExpiredKeyError) Error() string {
	return fmt.Sprintf("expired key %s", e.Key)
}

type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %s not found", e.Key)
}
