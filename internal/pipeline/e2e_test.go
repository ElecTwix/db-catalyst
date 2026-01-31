package pipeline

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen"
)

var update = flag.Bool("update", false, "update golden files")

func TestE2E(t *testing.T) {
	e2eDir := filepath.Join("testdata", "e2e")
	entries, err := os.ReadDir(e2eDir)
	if err != nil {
		t.Fatalf("failed to read e2e directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			caseDir := filepath.Join(e2eDir, entry.Name())
			runE2ETestCase(t, caseDir)
		})
	}
}

func runE2ETestCase(t *testing.T, caseDir string) {
	t.Helper()
	ctx := context.Background()
	// Create a temp workspace
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	// Copy fixtures to temp dir
	copyDir(t, tmpDir, caseDir)

	p := Pipeline{Env: Environment{Writer: &diskWriter{}}}
	summary, err := p.Run(ctx, RunOptions{
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("pipeline run failed: %v", err)
	}
	if len(summary.Diagnostics) > 0 {
		hasError := false
		for _, d := range summary.Diagnostics {
			t.Logf("diagnostic: %v", d)
			if d.Severity == 1 { // SeverityError
				hasError = true
			}
		}
		if hasError {
			t.Fatalf("encountered errors during pipeline run")
		}
	}

	goldenDir := filepath.Join(caseDir, "golden")
	if *update {
		updateGoldenFiles(t, tmpDir, goldenDir, caseDir, summary.Files)
		return
	}

	// Compare with golden files
	for _, file := range summary.Files {
		rel, err := filepath.Rel(filepath.Join(tmpDir, "gen"), file.Path)
		if err != nil {
			t.Fatalf("failed to get relative path: %v", err)
		}
		if strings.HasPrefix(rel, "..") {
			t.Fatalf("golden path outside golden dir: %s", rel)
		}
		goldenPath := filepath.Join(goldenDir, rel)
		goldenContent, err := os.ReadFile(filepath.Clean(goldenPath))
		if err != nil {
			t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
		}

		if string(file.Content) != string(goldenContent) {
			t.Errorf("file %s does not match golden", rel)
		}
	}
}

func updateGoldenFiles(t *testing.T, tmpDir, goldenDir, caseDir string, files []codegen.File) {
	t.Helper()
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("failed to clear golden dir: %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0750); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}
	for _, file := range files {
		rel, err := filepath.Rel(filepath.Join(tmpDir, "gen"), file.Path)
		if err != nil {
			t.Fatalf("failed to get relative path: %v", err)
		}
		dst := filepath.Join(goldenDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
			t.Fatalf("failed to create dst dir: %v", err)
		}
		if err := os.WriteFile(dst, file.Content, 0600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
	}
	t.Logf("updated golden files for %s", caseDir)
}

type diskWriter struct{}

func (w *diskWriter) WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
