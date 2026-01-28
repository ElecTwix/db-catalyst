package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestHooks_Chain(t *testing.T) {
	t.Run("chains two hooks", func(t *testing.T) {
		var calls []string

		h1 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				calls = append(calls, "h1")
				return nil
			},
		}

		h2 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				calls = append(calls, "h2")
				return nil
			},
		}

		chained := h1.Chain(h2)
		err := chained.BeforeParse(context.Background(), nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(calls) != 2 || calls[0] != "h1" || calls[1] != "h2" {
			t.Errorf("calls = %v, want [h1 h2]", calls)
		}
	})

	t.Run("first error stops chain", func(t *testing.T) {
		h1 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				return errors.New("h1 error")
			},
		}

		var h2Called bool
		h2 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				h2Called = true
				return nil
			},
		}

		chained := h1.Chain(h2)
		err := chained.BeforeParse(context.Background(), nil)

		if err == nil || err.Error() != "h1 error" {
			t.Errorf("error = %v, want 'h1 error'", err)
		}

		if h2Called {
			t.Error("h2 should not have been called")
		}
	})

	t.Run("nil first hook", func(t *testing.T) {
		var called bool
		h2 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				called = true
				return nil
			},
		}

		chained := NoHooks().Chain(h2)
		err := chained.BeforeParse(context.Background(), nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Error("h2 should have been called")
		}
	})

	t.Run("nil second hook", func(t *testing.T) {
		var called bool
		h1 := Hooks{
			BeforeParse: func(ctx context.Context, paths []string) error {
				called = true
				return nil
			},
		}

		chained := h1.Chain(NoHooks())
		err := chained.BeforeParse(context.Background(), nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Error("h1 should have been called")
		}
	})
}

func TestPipeline_Run_WithHooks(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	var hookCalls []string

	hooks := Hooks{
		BeforeParse: func(ctx context.Context, paths []string) error {
			hookCalls = append(hookCalls, "BeforeParse")
			return nil
		},
		AfterParse: func(ctx context.Context, catalog *model.Catalog) error {
			hookCalls = append(hookCalls, "AfterParse")
			return nil
		},
		BeforeAnalyze: func(ctx context.Context, paths []string) error {
			hookCalls = append(hookCalls, "BeforeAnalyze")
			return nil
		},
		AfterAnalyze: func(ctx context.Context, analyses []analyzer.Result) error {
			hookCalls = append(hookCalls, "AfterAnalyze")
			return nil
		},
		BeforeGenerate: func(ctx context.Context, analyses []analyzer.Result) error {
			hookCalls = append(hookCalls, "BeforeGenerate")
			return nil
		},
		AfterGenerate: func(ctx context.Context, files []codegen.File) error {
			hookCalls = append(hookCalls, "AfterGenerate")
			return nil
		},
		BeforeWrite: func(ctx context.Context, files []codegen.File) error {
			hookCalls = append(hookCalls, "BeforeWrite")
			return nil
		},
		AfterWrite: func(ctx context.Context, summary Summary) error {
			hookCalls = append(hookCalls, "AfterWrite")
			return nil
		},
	}

	pipeline := &Pipeline{
		Env: Environment{
			Logger: logging.NewSlogAdapter(slog.Default()),
			Writer: &MemoryWriter{},
		},
		Hooks: hooks,
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
		DryRun:     true,
	}

	_, err := pipeline.Run(ctx, opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expected := []string{
		"BeforeParse",
		"AfterParse",
		"BeforeAnalyze",
		"AfterAnalyze",
		"BeforeGenerate",
		"AfterGenerate",
		"BeforeWrite",
		"AfterWrite",
	}

	if len(hookCalls) != len(expected) {
		t.Errorf("hookCalls = %v, want %v", hookCalls, expected)
	}

	for i, call := range expected {
		if i >= len(hookCalls) || hookCalls[i] != call {
			t.Errorf("hookCalls[%d] = %v, want %v", i, hookCalls[i], call)
		}
	}
}

func TestPipeline_Run_HookError(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	hooks := Hooks{
		BeforeParse: func(ctx context.Context, paths []string) error {
			return errors.New("hook error")
		},
	}

	pipeline := &Pipeline{
		Env: Environment{
			Logger: logging.NewSlogAdapter(slog.Default()),
			Writer: &MemoryWriter{},
		},
		Hooks: hooks,
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	}

	_, err := pipeline.Run(ctx, opts)
	if err == nil {
		t.Fatal("expected error from hook")
	}

	if !strings.Contains(err.Error(), "hook error") {
		t.Errorf("error = %v, want to contain 'hook error'", err)
	}
}
