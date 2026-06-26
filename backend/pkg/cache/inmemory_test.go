package cache

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	c := New()

	c.Set("key1", []byte("value1"), 60)

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(val) != "value1" {
		t.Fatalf("expected 'value1', got '%s'", string(val))
	}
}

func TestExpiry(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 1) // 1 second TTL

	time.Sleep(1500 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestNoExpiry(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 0) // No expiry

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(val) != "value1" {
		t.Fatalf("expected 'value1', got '%s'", string(val))
	}
}

func TestDelete(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 60)

	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestMaxItems(t *testing.T) {
	c := New(WithMaxItems(3))
	c.Set("key1", []byte("value1"), 60)
	c.Set("key2", []byte("value2"), 60)
	c.Set("key3", []byte("value3"), 60)
	c.Set("key4", []byte("value4"), 60) // Should evict one

	if c.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", c.Len())
	}
}

func TestEvictCallback(t *testing.T) {
	evicted := make(map[string]bool)
	c := New(
		WithMaxItems(2),
		WithEvictCallback(func(key string, value []byte) {
			evicted[key] = true
		}),
	)
	c.Set("key1", []byte("value1"), 60)
	c.Set("key2", []byte("value2"), 60)
	c.Set("key3", []byte("value3"), 60) // Should evict key1

	if len(evicted) != 1 {
		t.Fatalf("expected 1 eviction, got %d", len(evicted))
	}
}

func TestClear(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 60)
	c.Set("key2", []byte("value2"), 60)
	c.Clear()

	if c.Len() != 0 {
		t.Fatal("expected cache to be empty")
	}
}

func TestHas(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 60)

	if !c.Has("key1") {
		t.Fatal("expected key1 to exist")
	}
	if c.Has("key2") {
		t.Fatal("expected key2 to not exist")
	}
}

func TestKeys(t *testing.T) {
	c := New()
	c.Set("key1", []byte("value1"), 60)
	c.Set("key2", []byte("value2"), 60)
	keys := c.Keys()

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestStats(t *testing.T) {
	c := New(WithMaxItems(100))
	c.Set("key1", []byte("value1"), 60)
	c.Set("key2", []byte("value2"), 60)
	stats := c.Stats()

	if stats.Items != 2 {
		t.Fatalf("expected 2 items, got %d", stats.Items)
	}
	if stats.MaxItems != 100 {
		t.Fatalf("expected max 100, got %d", stats.MaxItems)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New()
	done := make(chan bool)

	// Writer
	go func() {
		for i := 0; i < 1000; i++ {
			c.Set("key", []byte("value"), 60)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 1000; i++ {
			c.Get("key")
		}
		done <- true
	}()

	<-done
	<-done
}

func TestJSONValue(t *testing.T) {
	c := New()
	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	user := User{ID: "123", Name: "John"}
	data, _ := json.Marshal(user)
	c.Set("user:123", data, 60)

	val, ok := c.Get("user:123")
	if !ok {
		t.Fatal("expected key to exist")
	}
	var retrieved User
	json.Unmarshal(val, &retrieved)
	if retrieved.ID != "123" || retrieved.Name != "John" {
		t.Fatal("JSON roundtrip failed")
	}
}

// Benchmarks

func BenchmarkSet(b *testing.B) {
	c := New()
	value := []byte("test-value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("key", value, 60)
	}
}

func BenchmarkGet(b *testing.B) {
	c := New()
	c.Set("key", []byte("test-value"), 60)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkSetGet(b *testing.B) {
	c := New()
	value := []byte("test-value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("key", value, 60)
		c.Get("key")
	}
}

func BenchmarkConcurrentSetGet(b *testing.B) {
	c := New()
	value := []byte("test-value")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("key", value, 60)
			c.Get("key")
		}
	})
}
