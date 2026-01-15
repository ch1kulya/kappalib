package cache

import (
	"sync"
	"time"
)

type item struct {
	value      any
	expiration int64
}

type Cache struct {
	items map[string]item
	mutex sync.RWMutex
}

var C = &Cache{
	items: make(map[string]item),
}

func init() {
	C.startCleanup(5 * time.Minute)
}

func (c *Cache) startCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.deleteExpired()
		}
	}()
}

func (c *Cache) deleteExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	now := time.Now().UnixNano()
	for key, it := range c.items {
		if now >= it.expiration {
			delete(c.items, key)
		}
	}
}

func (c *Cache) Get(key string) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	it, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().UnixNano() >= it.expiration {
		return nil, false
	}

	return it.value, true
}

func (c *Cache) Set(key string, value any, duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = item{
		value:      value,
		expiration: time.Now().Add(duration).UnixNano(),
	}
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

func (c *Cache) GetOrFetch(key string, duration time.Duration, fetch func() (any, error)) (any, error) {
	if value, found := c.Get(key); found {
		return value, nil
	}

	value, err := fetch()
	if err != nil {
		return nil, err
	}

	c.Set(key, value, duration)
	return value, nil
}
