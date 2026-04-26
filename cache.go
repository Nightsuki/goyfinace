package yfinance

import (
	"sync"
	"time"
)

// Cache is the minimal interface for an HTTP-response cache. Implementations
// must be safe for concurrent use. The TTL is advisory — callers may treat
// zero as "no expiry" — and Set may be a no-op.
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
}

// MemoryCache is a simple in-process Cache. The zero value is unusable; use
// NewMemoryCache. Entries with TTL <= 0 never expire.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]memoryCacheEntry
}

type memoryCacheEntry struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache returns a ready-to-use MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: make(map[string]memoryCacheEntry)}
}

// Get returns the cached value if present and not expired.
func (m *MemoryCache) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !entry.expiration.IsZero() && time.Now().After(entry.expiration) {
		m.mu.Lock()
		delete(m.entries, key)
		m.mu.Unlock()
		return nil, false
	}
	out := make([]byte, len(entry.value))
	copy(out, entry.value)
	return out, true
}

// Set stores value under key. ttl <= 0 means no expiry.
func (m *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	cp := make([]byte, len(value))
	copy(cp, value)
	entry := memoryCacheEntry{value: cp}
	if ttl > 0 {
		entry.expiration = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.entries[key] = entry
	m.mu.Unlock()
}
