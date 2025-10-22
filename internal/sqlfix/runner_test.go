package sqlfix

import (
	"context"
	"testing"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestRunner_ExpandSelectStar(t *testing.T) {
	r := NewRunner()
	r.readFile = func(string) ([]byte, error) {
		return []byte("-- name: ListUsers :many\nSELECT *\nFROM users;\n"), nil
	}
	r.writeFile = func(string, []byte) error { return nil }

	catalog := model.NewCatalog()
	catalog.Tables["users"] = &model.Table{
		Name:    "users",
		Columns: []*model.Column{{Name: "id"}, {Name: "email"}},
	}
	r.SetCatalog(catalog, nil)

	report, out, err := r.rewriteFile(context.Background(), "queries.sql")
	if err != nil {
		t.Fatalf("rewriteFile: %v", err)
	}

	expected := "-- name: ListUsers :many\nSELECT id, email\nFROM users;\n"
	if string(out) != expected {
		t.Fatalf("unexpected output:\nwant %q\n got %q\nwarnings: %v", expected, string(out), report.Warnings)
	}
	if report.ExpandedStars != 1 {
		t.Fatalf("expected 1 expanded star, got %d", report.ExpandedStars)
	}
	if len(report.Added) != 0 {
		t.Fatalf("expected no aliases added, got %v", report.Added)
	}
}

func TestRunner_ExpandAliasStar(t *testing.T) {
	r := NewRunner()
	r.readFile = func(string) ([]byte, error) {
		return []byte("-- name: GetUser :one\nSELECT u.*\nFROM users AS u;\n"), nil
	}
	r.writeFile = func(string, []byte) error { return nil }

	catalog := model.NewCatalog()
	catalog.Tables["users"] = &model.Table{
		Name:    "users",
		Columns: []*model.Column{{Name: "id"}, {Name: "email"}},
	}
	r.SetCatalog(catalog, nil)

	report, out, err := r.rewriteFile(context.Background(), "queries.sql")
	if err != nil {
		t.Fatalf("rewriteFile: %v", err)
	}

	expected := "-- name: GetUser :one\nSELECT u.id, u.email\nFROM users AS u;\n"
	if string(out) != expected {
		t.Fatalf("unexpected output:\nwant %q\n got %q\nwarnings: %v", expected, string(out), report.Warnings)
	}
	if report.ExpandedStars != 1 {
		t.Fatalf("expected 1 expanded star, got %d", report.ExpandedStars)
	}
}

func TestRunner_ExpandJoinStar(t *testing.T) {
	r := NewRunner()
	r.readFile = func(string) ([]byte, error) {
		return []byte("-- name: ListJoined :many\nSELECT *\nFROM users\nJOIN accounts ON accounts.user_id = users.id;\n"), nil
	}
	r.writeFile = func(string, []byte) error { return nil }

	catalog := model.NewCatalog()
	catalog.Tables["users"] = &model.Table{
		Name:    "users",
		Columns: []*model.Column{{Name: "id"}, {Name: "email"}},
	}
	catalog.Tables["accounts"] = &model.Table{
		Name:    "accounts",
		Columns: []*model.Column{{Name: "user_id"}, {Name: "balance"}},
	}
	r.SetCatalog(catalog, nil)

	report, out, err := r.rewriteFile(context.Background(), "queries.sql")
	if err != nil {
		t.Fatalf("rewriteFile: %v", err)
	}

	expected := "-- name: ListJoined :many\nSELECT id, email, user_id, balance\nFROM users\nJOIN accounts ON accounts.user_id = users.id;\n"
	if string(out) != expected {
		t.Fatalf("unexpected output:\nwant %q\n got %q\nwarnings: %v", expected, string(out), report.Warnings)
	}
	if report.ExpandedStars != 1 {
		t.Fatalf("expected 1 expanded star, got %d", report.ExpandedStars)
	}
}

func TestRunner_UnresolvedAliasStar(t *testing.T) {
	r := NewRunner()
	r.readFile = func(string) ([]byte, error) {
		return []byte("-- name: ListUsers :many\nSELECT x.*\nFROM users AS u;\n"), nil
	}
	r.writeFile = func(string, []byte) error { return nil }

	catalog := model.NewCatalog()
	catalog.Tables["users"] = &model.Table{
		Name:    "users",
		Columns: []*model.Column{{Name: "id"}},
	}
	r.SetCatalog(catalog, nil)

	report, out, err := r.rewriteFile(context.Background(), "queries.sql")
	if err != nil {
		t.Fatalf("rewriteFile: %v", err)
	}

	expected := "-- name: ListUsers :many\nSELECT x.*\nFROM users AS u;\n"
	if string(out) != expected {
		t.Fatalf("expected output to remain unchanged\nwant %q\n got %q", expected, string(out))
	}
	if report.ExpandedStars != 0 {
		t.Fatalf("expected no expansions, got %d", report.ExpandedStars)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warning for unresolved alias")
	}
}
