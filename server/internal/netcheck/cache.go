package netcheck

import (
	"sync"
	"time"
)

// Cache holds recent reports so repeated lookups for the same target are
// served without re-querying.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	max     int
}

type cacheEntry struct {
	report  *Report
	expires time.Time
}

// NewCache builds a cache with the given entry lifetime and capacity.
func NewCache(ttl time.Duration, max int) *Cache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	if max <= 0 {
		max = 5000
	}
	return &Cache{entries: make(map[string]cacheEntry), ttl: ttl, max: max}
}

// Get returns a cached report if one is present and still fresh.
func (c *Cache) Get(key string) (*Report, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.report, true
}

// Set stores a report, evicting expired entries once the cache is full.
func (c *Cache) Set(key string, r *Report) {
	if c == nil || r == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expires) {
				delete(c.entries, k)
			}
		}

		if len(c.entries) >= c.max {
			c.entries = make(map[string]cacheEntry, c.max)
		}
	}

	c.entries[key] = cacheEntry{report: r, expires: time.Now().Add(c.ttl)}
}
