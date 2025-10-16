package pipeline

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/query/block"
)

type memoryWriter struct {
	writes map[string][]byte
	count  int
}

func (w *memoryWriter) WriteFile(path string, data []byte) error {
	if w.writes == nil {
		w.writes = make(map[string][]byte)
	}
	w.count++
	w.writes[path] = append([]byte(nil), data...)
	return nil
}

func TestPipelineDryRun(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
	if len(summary.Files) == 0 {
		t.Fatalf("Files = %v, want generated files", summary.Files)
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times during dry-run, want 0", writer.count)
	}
	if len(summary.Analyses) != 1 {
		t.Fatalf("Analyses = %d, want 1", len(summary.Analyses))
	}

	outPrefix := filepath.Join(filepath.Dir(configPath), "gen") + string(os.PathSeparator)
	for _, file := range summary.Files {
		if !strings.HasPrefix(file.Path, outPrefix) {
			t.Fatalf("file path %q does not reside under %q", file.Path, outPrefix)
		}
	}
}

func TestPipelineListQueries(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, ListQueries: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Files) != 0 {
		t.Fatalf("Files = %d, want none when listing", len(summary.Files))
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times when listing, want 0", writer.count)
	}
	if len(summary.Analyses) != 1 {
		t.Fatalf("Analyses = %d, want 1", len(summary.Analyses))
	}

	analysis := summary.Analyses[0]
	if analysis.Query.Block.Name != "ListUsers" {
		t.Fatalf("query name = %q, want ListUsers", analysis.Query.Block.Name)
	}
	if analysis.Query.Block.Command != block.CommandMany {
		t.Fatalf("command = %v, want CommandMany", analysis.Query.Block.Command)
	}
	if len(analysis.Params) != 0 {
		t.Fatalf("params = %v, want none", analysis.Params)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
}

func prepareFixtures(t *testing.T) string {
	t.Helper()
	src := filepath.Join("testdata")
	dst := t.TempDir()
	copyTree(t, dst, src)
	return filepath.Join(dst, "config.toml")
}

func copyTree(t *testing.T, dst, src string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("ReadDir %q: %v", src, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				t.Fatalf("MkdirAll %q: %v", dstPath, err)
			}
			copyTree(t, dstPath, srcPath)
			continue
		}
		copyFile(t, dstPath, srcPath)
	}
}

func copyFile(t *testing.T, dst, src string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %q: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("create %q: %v", dst, err)
	}
	defer func() {
		out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %q -> %q: %v", src, dst, err)
	}
}
