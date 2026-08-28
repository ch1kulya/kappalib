package cache

import (
	"errors"
	"sync"
	"sync/atomic"
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

func TestGetOrFetch_Singleflight(t *testing.T) {
	c := newTestCache()
	var (
		wg         sync.WaitGroup
		fetchCount atomic.Int32
		start      = make(chan struct{})
	)

	const n = 50
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start
			val, err := c.GetOrFetch("shared-key", time.Minute, func() (any, error) {
				fetchCount.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "data", nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val != "data" {
				t.Errorf("got %v, want data", val)
			}
		}()
	}

	close(start)
	wg.Wait()

	if count := fetchCount.Load(); count != 1 {
		t.Errorf("fetch called %d times, want 1", count)
	}
}

func TestGetOrFetch_SingleflightError(t *testing.T) {
	c := newTestCache()
	var (
		wg         sync.WaitGroup
		fetchCount atomic.Int32
		start      = make(chan struct{})
		fetchErr   = errors.New("boom")
	)

	const n = 20
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start
			_, err := c.GetOrFetch("error-key", time.Minute, func() (any, error) {
				fetchCount.Add(1)
				time.Sleep(10 * time.Millisecond)
				return nil, fetchErr
			})
			if !errors.Is(err, fetchErr) {
				t.Errorf("got err %v, want %v", err, fetchErr)
			}
		}()
	}

	close(start)
	wg.Wait()

	if count := fetchCount.Load(); count != 1 {
		t.Errorf("fetch called %d times, want 1", count)
	}

	val, err := c.GetOrFetch("error-key", time.Minute, func() (any, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("unexpected error on retry: %v", err)
	}
	if val != "recovered" {
		t.Errorf("got %v, want recovered", val)
	}
}

func TestGetOrFetch_SingleflightDistinctKeys(t *testing.T) {
	c := newTestCache()
	var (
		wg         sync.WaitGroup
		fetchCount atomic.Int32
		start      = make(chan struct{})
	)

	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for _, key := range keys {
		for range 5 {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				<-start
				val, err := c.GetOrFetch(k, time.Minute, func() (any, error) {
					fetchCount.Add(1)
					time.Sleep(10 * time.Millisecond)
					return "val-" + k, nil
				})
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if val != "val-"+k {
					t.Errorf("got %v, want val-%s", val, k)
				}
			}(key)
		}
	}

	close(start)
	wg.Wait()

	if count := fetchCount.Load(); count != int32(len(keys)) {
		t.Errorf("fetch called %d times, want %d", count, len(keys))
	}
}
