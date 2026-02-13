package block

import (
	"testing"
	"time"
)

func TestSliceWithCacheAnnotation(t *testing.T) {
	sql := `-- GetUser retrieves a user by ID.
-- @cache ttl=5m key=user:{id}
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- CreateUser inserts a new user.
-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES ($1, $2);

-- ListUsers retrieves all users.
-- @cache ttl=1h invalidate=users
-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;`

	blocks, err := Slice("test.sql", []byte(sql))
	if err != nil {
		t.Fatalf("Slice() error = %v", err)
	}

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}

	// Check GetUser block with cache annotation
	if blocks[0].Name != "GetUser" {
		t.Errorf("block[0].Name = %q, want GetUser", blocks[0].Name)
	}
	if blocks[0].Cache == nil {
		t.Fatal("expected GetUser to have cache annotation")
	}
	if blocks[0].Cache.TTL != 5*time.Minute {
		t.Errorf("GetUser cache TTL = %v, want 5m", blocks[0].Cache.TTL)
	}
	if blocks[0].Cache.KeyPattern != "user:{id}" {
		t.Errorf("GetUser cache KeyPattern = %q, want user:{id}", blocks[0].Cache.KeyPattern)
	}
	if len(blocks[0].Cache.Invalidate) != 0 {
		t.Errorf("GetUser cache Invalidate = %v, want empty", blocks[0].Cache.Invalidate)
	}

	// Check CreateUser block without cache annotation
	if blocks[1].Name != "CreateUser" {
		t.Errorf("block[1].Name = %q, want CreateUser", blocks[1].Name)
	}
	if blocks[1].Cache != nil {
		t.Errorf("expected CreateUser to have no cache annotation, got %v", blocks[1].Cache)
	}

	// Check ListUsers block with cache annotation
	if blocks[2].Name != "ListUsers" {
		t.Errorf("block[2].Name = %q, want ListUsers", blocks[2].Name)
	}
	if blocks[2].Cache == nil {
		t.Fatal("expected ListUsers to have cache annotation")
	}
	if blocks[2].Cache.TTL != 1*time.Hour {
		t.Errorf("ListUsers cache TTL = %v, want 1h", blocks[2].Cache.TTL)
	}
	if len(blocks[2].Cache.Invalidate) != 1 || blocks[2].Cache.Invalidate[0] != "users" {
		t.Errorf("ListUsers cache Invalidate = %v, want [users]", blocks[2].Cache.Invalidate)
	}
}

func TestSliceWithCacheAnnotationBare(t *testing.T) {
	sql := `-- GetUser retrieves a user by ID.
-- @cache
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;`

	blocks, err := Slice("test.sql", []byte(sql))
	if err != nil {
		t.Fatalf("Slice() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].Cache == nil {
		t.Fatal("expected cache annotation")
	}
	if blocks[0].Cache.TTL != 5*time.Minute {
		t.Errorf("cache TTL = %v, want 5m", blocks[0].Cache.TTL)
	}
}

func TestSliceWithMultipleAnnotations(t *testing.T) {
	sql := `-- GetUser retrieves a user by ID.
-- @param userID: uuid
-- @cache ttl=10m key=user:{userID}
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;`

	blocks, err := Slice("test.sql", []byte(sql))
	if err != nil {
		t.Fatalf("Slice() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	// Check param type override
	if len(blocks[0].ParamTypes) != 1 {
		t.Errorf("expected 1 param type override, got %d", len(blocks[0].ParamTypes))
	} else {
		if blocks[0].ParamTypes[0].ParamName != "userID" {
			t.Errorf("ParamTypes[0].ParamName = %q, want userID", blocks[0].ParamTypes[0].ParamName)
		}
		if blocks[0].ParamTypes[0].GoType != "uuid" {
			t.Errorf("ParamTypes[0].GoType = %q, want uuid", blocks[0].ParamTypes[0].GoType)
		}
	}

	// Check cache annotation
	if blocks[0].Cache == nil {
		t.Fatal("expected cache annotation")
	}
	if blocks[0].Cache.TTL != 10*time.Minute {
		t.Errorf("cache TTL = %v, want 10m", blocks[0].Cache.TTL)
	}
	if blocks[0].Cache.KeyPattern != "user:{userID}" {
		t.Errorf("cache KeyPattern = %q, want user:{userID}", blocks[0].Cache.KeyPattern)
	}
}
