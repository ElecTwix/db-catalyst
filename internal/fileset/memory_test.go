package fileset

import (
	"errors"
	"testing"
)

func TestMemoryResolver_Resolve(t *testing.T) {
	files := map[string][]byte{
		"schema.sql":      []byte("CREATE TABLE users (id INTEGER);"),
		"queries.sql":     []byte("SELECT * FROM users;"),
		"test.txt":        []byte("not sql"),
		"nested/nest.sql": []byte("SELECT 1;"),
	}

	r := NewMemoryResolver("/test", files)

	t.Run("exact match", func(t *testing.T) {
		paths, err := r.Resolve([]string{"schema.sql"})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if len(paths) != 1 || paths[0] != "schema.sql" {
			t.Errorf("Resolve() = %v, want [schema.sql]", paths)
		}
	})

	t.Run("glob pattern *.sql", func(t *testing.T) {
		paths, err := r.Resolve([]string{"*.sql"})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("Resolve() returned %d paths, want 2", len(paths))
		}
	})

	t.Run("multiple patterns", func(t *testing.T) {
		paths, err := r.Resolve([]string{"schema.sql", "queries.sql"})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("Resolve() returned %d paths, want 2", len(paths))
		}
	})

	t.Run("no patterns", func(t *testing.T) {
		_, err := r.Resolve([]string{})
		if !errors.Is(err, ErrNoPatterns) {
			t.Errorf("Resolve() error = %v, want ErrNoPatterns", err)
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, err := r.Resolve([]string{"missing.sql"})
		if err == nil {
			t.Error("expected error for non-matching pattern")
		}
	})
}

func TestMemoryResolver_ReadFile(t *testing.T) {
	files := map[string][]byte{
		"test.sql": []byte("SELECT 1;"),
	}

	r := NewMemoryResolver("/test", files)

	t.Run("existing file", func(t *testing.T) {
		content, err := r.ReadFile("test.sql")
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(content) != "SELECT 1;" {
			t.Errorf("ReadFile() = %q, want %q", string(content), "SELECT 1;")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := r.ReadFile("missing.sql")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestMemoryResolver_AddRemoveFile(t *testing.T) {
	r := NewMemoryResolver("/test", map[string][]byte{})

	t.Run("add file", func(t *testing.T) {
		r.AddFile("new.sql", []byte("CREATE TABLE t (id INT);"))

		if r.FileCount() != 1 {
			t.Errorf("FileCount() = %d, want 1", r.FileCount())
		}

		content, _ := r.ReadFile("new.sql")
		if string(content) != "CREATE TABLE t (id INT);" {
			t.Error("file content mismatch")
		}
	})

	t.Run("remove file", func(t *testing.T) {
		r.RemoveFile("new.sql")

		if r.FileCount() != 0 {
			t.Errorf("FileCount() = %d, want 0", r.FileCount())
		}
	})
}
