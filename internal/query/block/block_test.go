package block

import (
	"strings"
	"testing"
)

func TestSliceMultiBlock(t *testing.T) {
	src := `-- name: GetUser :one
SELECT id, name FROM users WHERE id = ?;

-- Retrieves all users
-- Another line
-- name: ListUsers :many
SELECT id, name
FROM users
ORDER BY name;
`
	blocks, err := Slice("query/users.sql", []byte(src))
	if err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	first := blocks[0]
	if first.Name != "GetUser" {
		t.Errorf("unexpected first block name %q", first.Name)
	}
	if first.Command != CommandOne {
		t.Errorf("expected first command CommandOne, got %v", first.Command)
	}
	expectedSQL := "SELECT id, name FROM users WHERE id = ?;"
	if first.SQL != expectedSQL {
		t.Errorf("unexpected SQL for first block:\nwant %q\n got %q", expectedSQL, first.SQL)
	}
	if strings.Contains(first.SQL, "Retrieves") {
		t.Errorf("doc comments leaked into SQL: %q", first.SQL)
	}

	second := blocks[1]
	if second.Name != "ListUsers" {
		t.Errorf("unexpected second block name %q", second.Name)
	}
	if second.Command != CommandMany {
		t.Errorf("expected second command CommandMany, got %v", second.Command)
	}
	expectedDoc := "Retrieves all users\nAnother line"
	if second.Doc != expectedDoc {
		t.Errorf("unexpected doc: want %q got %q", expectedDoc, second.Doc)
	}
	if second.Line != 6 {
		t.Errorf("expected second block line 6, got %d", second.Line)
	}
	if second.Column != 1 {
		t.Errorf("expected second block column 1, got %d", second.Column)
	}
}

func TestSliceInvalidCommand(t *testing.T) {
	src := `-- name: BadBlock :bogus
SELECT 1;
`
	_, err := Slice("query/users.sql", []byte(src))
	if err == nil {
		t.Fatalf("expected error for invalid command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestSliceMissingCommand(t *testing.T) {
	src := `-- name: Missing
SELECT 1;
`
	_, err := Slice("query/users.sql", []byte(src))
	if err == nil {
		t.Fatalf("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "missing command") {
		t.Fatalf("expected missing command error, got %v", err)
	}
}

func TestSliceSQLBeforeMarker(t *testing.T) {
	src := "SELECT 1;\n"
	_, err := Slice("query/users.sql", []byte(src))
	if err == nil {
		t.Fatalf("expected error for SQL before first marker")
	}
	if !strings.Contains(err.Error(), "before block marker") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCommand(t *testing.T) {
	cases := map[string]Command{
		":one":        CommandOne,
		":many":       CommandMany,
		":exec":       CommandExec,
		":execresult": CommandExecResult,
	}
	for tag, want := range cases {
		got, ok := ParseCommand(tag)
		if !ok {
			t.Fatalf("expected %s to parse", tag)
		}
		if got != want {
			t.Fatalf("tag %s parsed as %v, want %v", tag, got, want)
		}
	}
	if _, ok := ParseCommand(":nope"); ok {
		t.Fatalf("expected :nope to be invalid")
	}
}
