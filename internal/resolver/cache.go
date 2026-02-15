package resolver

import (
	"sync"
	"time"
)

const defaultCacheTTL = 5 * time.Minute

// cacheEntry holds a cached Vault KV response with its expiration time.
type cacheEntry struct {
	data      map[string]string
	expiresAt time.Time
}

// Cache is a thread-safe in-memory cache for Vault KV v2 responses keyed
// by Vault path. Entries expire after the configured TTL.
type Cache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cacheEntry
}

// NewCache creates a new Cache with the given TTL. If ttl is zero or
// negative, the default of 5 minutes is used.
func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}

	return &Cache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
	}
}

// Get returns the cached KV data for the given path and true if found and
// not expired. Returns nil, false on a miss or expiration.
func (c *Cache) Get(path string) (map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}

	// Return a copy to prevent mutation of cached data.
	return copyMap(entry.data), true
}

// Set stores KV data for the given path. The data is copied to prevent
// external mutation of cached values.
func (c *Cache) Set(path string, data map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = cacheEntry{
		data:      copyMap(data),
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]cacheEntry)
}

// copyMap returns a shallow copy of a string map.
func copyMap(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}

	return cp
}
