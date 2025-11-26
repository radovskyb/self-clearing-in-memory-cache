package cache

import (
	"log"
	"reflect"
	"sync"
)

// Cache is a simple in-memory cache. Safe for concurrent use and rotates when maxCacheSize is hit.
type Cache struct {
	mu             sync.RWMutex
	items          map[string]any
	totalCacheSize int64
	maxCacheSize   int64
}

// New creates a new in-memory cache.
func New(maxCacheSize int64) *Cache {
	return &Cache{
		maxCacheSize: maxCacheSize,
		items:        make(map[string]any),
	}
}

// Get retrieves an item from the cache.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	return item, found
}

func estimateItemSize(value any) int64 {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return int64(v.Len())
		}
	case reflect.String:
		return int64(v.Len())
	}

	// Default minimal size estimate (Adjust this based on either config or use)
	return 32
}

// Set adds an item to the cache, replacing any existing item.
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keySize := int64(len(key))
	newItemSize := estimateItemSize(value)

	if oldValue, found := c.items[key]; found {
		oldItemSize := estimateItemSize(oldValue)
		c.totalCacheSize -= oldItemSize
	} else {
		c.totalCacheSize += keySize
	}

	c.totalCacheSize += newItemSize

	c.items[key] = value

	c.checkCurrentSize()
}

// Delete removes an item from the cache and updates the size.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if oldValue, found := c.items[key]; found {
		c.totalCacheSize -= int64(len(key))
		c.totalCacheSize -= estimateItemSize(oldValue)

		delete(c.items, key)
		c.checkCurrentSize()
	}
}

func (c *Cache) checkCurrentSize() {
	log.Printf("current cache size: %d bytes", c.totalCacheSize)

	if c.totalCacheSize > c.maxCacheSize {
		log.Printf("cache size exceeded limit (%d bytes). clearing...", c.totalCacheSize)

		// This is a good place if you want to chuck in some handling. (I've sent admin notifications here which works alright)
		// You'd run this in a goroutine, since this would likely be a "long" running process.
		// go func(curSize int64) {}(c.totalCacheSize)

		// Clear the cache
		//
		// No mutex is locked here since we're only calling this func where c.mu is already locked.
		c.items = make(map[string]any)
		c.totalCacheSize = 0

		log.Println("cache successfully cleared. size reset to 0 bytes.")
	}
}
