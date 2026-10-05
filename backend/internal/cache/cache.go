// Package cache is a tiny in-process replacement for the file cache the
// Laravel app used for currencies, languages and settings.
package cache

import (
	"sync"
	"time"
)

// Value memoises one computed value for a TTL (zero = forever) until Flush.
type Value[T any] struct {
	mu      sync.Mutex
	val     T
	ok      bool
	expires time.Time
}

func (c *Value[T]) Get(ttl time.Duration, load func() (T, error)) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ok && (c.expires.IsZero() || time.Now().Before(c.expires)) {
		return c.val, nil
	}

	val, err := load()
	if err != nil {
		var zero T
		return zero, err
	}

	c.val, c.ok = val, true
	if ttl > 0 {
		c.expires = time.Now().Add(ttl)
	} else {
		c.expires = time.Time{}
	}
	return val, nil
}

func (c *Value[T]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero T
	c.val, c.ok = zero, false
}
