package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRun(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}

	expected := filepath.Join(filepath.Dir(configPath), "gen", "querier.gen.go")
	if !strings.Contains(stdout.String(), expected) {
		t.Fatalf("stdout %q missing generated file %q", stdout.String(), expected)
	}
}

func TestRunListQueries(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "ListUsers") {
		t.Fatalf("stdout %q missing query name", out)
	}
	if !strings.Contains(out, ":many") {
		t.Fatalf("stdout %q missing command tag", out)
	}
	if !strings.Contains(out, "params: none") {
		t.Fatalf("stdout %q missing params summary", out)
	}
}

func prepareCmdFixtures(t *testing.T) string {
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
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %q -> %q: %v", src, dst, err)
	}
}
