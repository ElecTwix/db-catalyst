package pipeline

import (
	"fmt"
	"testing"
)

func TestMemoryWriter_WriteFile(t *testing.T) {
	w := &MemoryWriter{}

	t.Run("write and retrieve", func(t *testing.T) {
		err := w.WriteFile("test.go", []byte("package main"))
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		data, ok := w.GetFile("test.go")
		if !ok {
			t.Fatal("GetFile() returned false")
		}
		if string(data) != "package main" {
			t.Errorf("GetFile() = %q, want %q", string(data), "package main")
		}
	})

	t.Run("overwrite existing", func(t *testing.T) {
		_ = w.WriteFile("test.go", []byte("first"))
		_ = w.WriteFile("test.go", []byte("second"))

		data, _ := w.GetFile("test.go")
		if string(data) != "second" {
			t.Error("expected file to be overwritten")
		}
	})

	t.Run("has file", func(t *testing.T) {
		_ = w.WriteFile("exists.go", []byte(""))

		if !w.HasFile("exists.go") {
			t.Error("HasFile() returned false for existing file")
		}
		if w.HasFile("missing.go") {
			t.Error("HasFile() returned true for missing file")
		}
	})

	t.Run("file count", func(t *testing.T) {
		w.Clear()
		if w.FileCount() != 0 {
			t.Errorf("FileCount() = %d, want 0", w.FileCount())
		}

		_ = w.WriteFile("a.go", []byte("a"))
		_ = w.WriteFile("b.go", []byte("b"))

		if w.FileCount() != 2 {
			t.Errorf("FileCount() = %d, want 2", w.FileCount())
		}
	})

	t.Run("clear", func(t *testing.T) {
		_ = w.WriteFile("test.go", []byte("data"))
		w.Clear()

		if w.HasFile("test.go") {
			t.Error("HasFile() returned true after Clear()")
		}
	})
}

func TestMemoryWriter_Concurrent(t *testing.T) {
	w := &MemoryWriter{}

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			path := fmt.Sprintf("file%d.go", n)
			_ = w.WriteFile(path, []byte("data"))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if w.FileCount() != 10 {
		t.Errorf("FileCount() = %d, want 10", w.FileCount())
	}
}
