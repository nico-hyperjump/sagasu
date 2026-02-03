package embedding

import (
	"container/list"
	"sync"
)

// EmbeddingCache is an LRU cache for embeddings keyed by text.
type EmbeddingCache struct {
	capacity int
	cache    map[string]*list.Element
	lru      *list.List
	mu       sync.RWMutex
}

type cacheEntry struct {
	key   string
	value []float32
}

// NewEmbeddingCache creates a new cache with the given capacity.
func NewEmbeddingCache(capacity int) *EmbeddingCache {
	return &EmbeddingCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get returns the cached embedding for key if present.
func (c *EmbeddingCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry).value, true
	}
	return nil, false
}

// Set stores the embedding for key, evicting the oldest entry if at capacity.
func (c *EmbeddingCache) Set(key string, value []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	entry := &cacheEntry{key: key, value: value}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	if c.lru.Len() > c.capacity {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.cache, oldest.Value.(*cacheEntry).key)
		}
	}
}
