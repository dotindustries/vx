package resolver

import (
	"sync"
	"testing"
	"time"
)

func TestCache_HitAndMiss(t *testing.T) {
	c := NewCache(time.Minute)

	data := map[string]string{"url": "postgres://localhost"}
	c.Set("dev/database", data)

	got, ok := c.Get("dev/database")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}

	if got["url"] != "postgres://localhost" {
		t.Errorf("cached value = %q, want %q", got["url"], "postgres://localhost")
	}

	_, ok = c.Get("staging/database")
	if ok {
		t.Error("expected cache miss for unknown path")
	}
}

func TestCache_ReturnsCopy(t *testing.T) {
	c := NewCache(time.Minute)

	original := map[string]string{"key": "value"}
	c.Set("path", original)

	got, _ := c.Get("path")
	got["key"] = "mutated"

	got2, _ := c.Get("path")
	if got2["key"] != "value" {
		t.Error("cache entry was mutated through returned map")
	}
}

func TestCache_SetDoesNotRetainReference(t *testing.T) {
	c := NewCache(time.Minute)

	input := map[string]string{"key": "original"}
	c.Set("path", input)

	input["key"] = "mutated"

	got, _ := c.Get("path")
	if got["key"] != "original" {
		t.Error("cache entry was mutated through input map reference")
	}
}

func TestCache_Expiry(t *testing.T) {
	c := NewCache(10 * time.Millisecond)

	c.Set("dev/database", map[string]string{"url": "localhost"})

	_, ok := c.Get("dev/database")
	if !ok {
		t.Fatal("expected cache hit before expiry")
	}

	time.Sleep(20 * time.Millisecond)

	_, ok = c.Get("dev/database")
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestCache_Clear(t *testing.T) {
	c := NewCache(time.Minute)

	c.Set("path1", map[string]string{"a": "1"})
	c.Set("path2", map[string]string{"b": "2"})
	c.Clear()

	_, ok1 := c.Get("path1")
	_, ok2 := c.Get("path2")

	if ok1 || ok2 {
		t.Error("expected all entries cleared")
	}
}

func TestCache_DefaultTTL(t *testing.T) {
	c := NewCache(0)
	if c.ttl != defaultCacheTTL {
		t.Errorf("default TTL = %v, want %v", c.ttl, defaultCacheTTL)
	}

	c2 := NewCache(-1 * time.Second)
	if c2.ttl != defaultCacheTTL {
		t.Errorf("negative TTL should use default, got %v", c2.ttl)
	}
}

func TestCache_ThreadSafety(t *testing.T) {
	c := NewCache(time.Minute)

	var wg sync.WaitGroup
	const goroutines = 100

	wg.Add(goroutines * 3)

	for i := range goroutines {
		path := "path"
		data := map[string]string{"key": string(rune('a' + i%26))}

		go func() {
			defer wg.Done()
			c.Set(path, data)
		}()

		go func() {
			defer wg.Done()
			c.Get(path)
		}()

		go func() {
			defer wg.Done()
			if i%10 == 0 {
				c.Clear()
			}
		}()
	}

	wg.Wait()
}
