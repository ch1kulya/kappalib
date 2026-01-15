package cache

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func newTestCache() *Cache {
	return &Cache{
		items: make(map[string]item),
	}
}

func TestSetAndGet(t *testing.T) {
	c := newTestCache()

	c.Set("key", "value", time.Minute)

	got, found := c.Get("key")
	if !found {
		t.Fatal("expected key to be found")
	}
	if got != "value" {
		t.Errorf("got %v, want %v", got, "value")
	}
}

func TestGetNonExistent(t *testing.T) {
	c := newTestCache()

	_, found := c.Get("nonexistent")
	if found {
		t.Error("expected key not to be found")
	}
}

func TestExpiration(t *testing.T) {
	c := newTestCache()

	c.Set("key", "value", 50*time.Millisecond)

	if _, found := c.Get("key"); !found {
		t.Fatal("key should exist before expiration")
	}

	time.Sleep(60 * time.Millisecond)

	if _, found := c.Get("key"); found {
		t.Error("key should not be found after expiration")
	}
}

func TestDelete(t *testing.T) {
	c := newTestCache()

	c.Set("key", "value", time.Minute)
	c.Delete("key")

	if _, found := c.Get("key"); found {
		t.Error("key should not be found after deletion")
	}
}

func TestDeleteExpired(t *testing.T) {
	c := newTestCache()

	c.Set("expired", "value", 10*time.Millisecond)
	c.Set("valid", "value", time.Minute)

	time.Sleep(20 * time.Millisecond)
	c.deleteExpired()

	c.mutex.RLock()
	_, expiredExists := c.items["expired"]
	_, validExists := c.items["valid"]
	c.mutex.RUnlock()

	if expiredExists {
		t.Error("expired item should be removed from map")
	}
	if !validExists {
		t.Error("valid item should remain in map")
	}
}

func TestGetOrFetch_CacheHit(t *testing.T) {
	c := newTestCache()
	c.Set("key", "cached", time.Minute)

	fetchCalled := false
	got, err := c.GetOrFetch("key", time.Minute, func() (any, error) {
		fetchCalled = true
		return "fetched", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCalled {
		t.Error("fetch should not be called on cache hit")
	}
	if got != "cached" {
		t.Errorf("got %v, want %v", got, "cached")
	}
}

func TestGetOrFetch_CacheMiss(t *testing.T) {
	c := newTestCache()

	got, err := c.GetOrFetch("key", time.Minute, func() (any, error) {
		return "fetched", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fetched" {
		t.Errorf("got %v, want %v", got, "fetched")
	}

	cached, found := c.Get("key")
	if !found {
		t.Error("value should be cached after fetch")
	}
	if cached != "fetched" {
		t.Errorf("cached value = %v, want %v", cached, "fetched")
	}
}

func TestGetOrFetch_FetchError(t *testing.T) {
	c := newTestCache()
	fetchErr := errors.New("fetch failed")

	_, err := c.GetOrFetch("key", time.Minute, func() (any, error) {
		return nil, fetchErr
	})

	if !errors.Is(err, fetchErr) {
		t.Errorf("got error %v, want %v", err, fetchErr)
	}

	if _, found := c.Get("key"); found {
		t.Error("failed fetch should not cache value")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := newTestCache()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(3)

		go func(i int) {
			defer wg.Done()
			c.Set("key", i, time.Minute)
		}(i)

		go func() {
			defer wg.Done()
			c.Get("key")
		}()

		go func() {
			defer wg.Done()
			c.Delete("key")
		}()
	}

	wg.Wait()
}
