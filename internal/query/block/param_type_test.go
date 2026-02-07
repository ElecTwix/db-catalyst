package block

import (
	"testing"
)

func TestParamTypeParsing(t *testing.T) {
	src := []byte(`-- @param userID: uuid.UUID
-- @param email: custom.Email
-- name: GetUser :one
SELECT * FROM users WHERE id = :userID AND email = :email;
`)

	blocks, err := Slice("test.sql", src)
	if err != nil {
		t.Fatalf("Slice failed: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	blk := blocks[0]
	if len(blk.ParamTypes) != 2 {
		t.Fatalf("expected 2 param types, got %d: %+v", len(blk.ParamTypes), blk.ParamTypes)
	}

	if blk.ParamTypes[0].ParamName != "userID" || blk.ParamTypes[0].GoType != "uuid.UUID" {
		t.Errorf("expected first param type {userID uuid.UUID}, got %+v", blk.ParamTypes[0])
	}

	if blk.ParamTypes[1].ParamName != "email" || blk.ParamTypes[1].GoType != "custom.Email" {
		t.Errorf("expected second param type {email custom.Email}, got %+v", blk.ParamTypes[1])
	}
}

func TestParamTypeParsingWithColon(t *testing.T) {
	src := []byte(`-- Get a user by ID
-- @param userID: uuid.UUID
-- name: GetUser :one
SELECT * FROM users WHERE id = :userID;
`)

	blocks, err := Slice("test.sql", src)
	if err != nil {
		t.Fatalf("Slice failed: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	blk := blocks[0]
	if len(blk.ParamTypes) != 1 {
		t.Fatalf("expected 1 param type, got %d: %+v", len(blk.ParamTypes), blk.ParamTypes)
	}

	if blk.ParamTypes[0].ParamName != "userID" || blk.ParamTypes[0].GoType != "uuid.UUID" {
		t.Errorf("expected param type {userID uuid.UUID}, got %+v", blk.ParamTypes[0])
	}

	// Check that doc doesn't include @param lines
	if blk.Doc != "Get a user by ID" {
		t.Errorf("expected doc 'Get a user by ID', got %q", blk.Doc)
	}
}
