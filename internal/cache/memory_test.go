package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_GetSet(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	t.Run("set and get", func(t *testing.T) {
		c.Set(ctx, "key", "value", time.Hour)

		val, ok := c.Get(ctx, "key")
		if !ok {
			t.Fatal("expected key to exist")
		}
		if val != "value" {
			t.Errorf("Get() = %v, want %v", val, "value")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := c.Get(ctx, "missing")
		if ok {
			t.Error("expected key to not exist")
		}
	})

	t.Run("expired entry", func(t *testing.T) {
		c.Set(ctx, "expired", "value", -time.Hour) // already expired

		_, ok := c.Get(ctx, "expired")
		if ok {
			t.Error("expected expired key to not exist")
		}
	})
}

func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	c.Set(ctx, "key", "value", time.Hour)
	c.Delete(ctx, "key")

	_, ok := c.Get(ctx, "key")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	c.Set(ctx, "key1", "value1", time.Hour)
	c.Set(ctx, "key2", "value2", time.Hour)

	c.Clear(ctx)

	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
}

func TestMemoryCache_Cleanup(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	c.Set(ctx, "valid", "value", time.Hour)
	c.Set(ctx, "expired", "value", -time.Second)

	if c.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", c.Len())
	}

	c.Cleanup()

	if c.Len() != 1 {
		t.Errorf("Len() = %d, want 1", c.Len())
	}

	_, ok := c.Get(ctx, "valid")
	if !ok {
		t.Error("expected valid key to exist")
	}
}

func TestComputeKey(t *testing.T) {
	key1 := ComputeKey([]byte("content"))
	key2 := ComputeKey([]byte("content"))
	key3 := ComputeKey([]byte("different"))

	if key1 != key2 {
		t.Error("same content should produce same key")
	}

	if key1 == key3 {
		t.Error("different content should produce different key")
	}

	if len(key1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("key length = %d, want 32", len(key1))
	}
}

func TestComputeKeyWithPrefix(t *testing.T) {
	key := ComputeKeyWithPrefix("schema", []byte("content"))

	if len(key) < 8 { // at least "schema:" + something
		t.Error("key too short")
	}

	if key[:7] != "schema:" {
		t.Errorf("key prefix = %q, want 'schema:'", key[:7])
	}
}
