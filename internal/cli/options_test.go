package cli

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if opts.ConfigPath != "db-catalyst.toml" {
		t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "db-catalyst.toml")
	}
	if opts.Out != "" {
		t.Fatalf("Out = %q, want empty", opts.Out)
	}
	if opts.DryRun {
		t.Fatalf("DryRun = true, want false")
	}
	if opts.ListQueries {
		t.Fatalf("ListQueries = true, want false")
	}
	if opts.StrictConfig {
		t.Fatalf("StrictConfig = true, want false")
	}
	if opts.Verbose {
		t.Fatalf("Verbose = true, want false")
	}
	if len(opts.Args) != 0 {
		t.Fatalf("Args = %v, want empty slice", opts.Args)
	}
}

func TestParseOverrides(t *testing.T) {
	args := []string{
		"--config", "project.toml",
		"--out", "build",
		"--dry-run",
		"--list-queries",
		"--strict-config",
		"-v",
		"extra",
	}

	opts, err := Parse(args)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got, want := opts.ConfigPath, "project.toml"; got != want {
		t.Fatalf("ConfigPath = %q, want %q", got, want)
	}
	if got, want := opts.Out, "build"; got != want {
		t.Fatalf("Out = %q, want %q", got, want)
	}
	if !opts.DryRun {
		t.Fatalf("DryRun = false, want true")
	}
	if !opts.ListQueries {
		t.Fatalf("ListQueries = false, want true")
	}
	if !opts.StrictConfig {
		t.Fatalf("StrictConfig = false, want true")
	}
	if !opts.Verbose {
		t.Fatalf("Verbose = false, want true")
	}
	if len(opts.Args) != 1 || opts.Args[0] != "extra" {
		t.Fatalf("Args = %v, want [extra]", opts.Args)
	}
}

func TestParseInvalidFlag(t *testing.T) {
	_, err := Parse([]string{"--unknown"})
	if err == nil {
		t.Fatalf("Parse expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "Usage of db-catalyst") {
		t.Fatalf("error = %q, want usage string", err.Error())
	}
	if errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error unexpectedly wraps flag.ErrHelp")
	}
}

func TestUsage(t *testing.T) {
	fs := flag.NewFlagSet("db-catalyst", flag.ContinueOnError)
	fs.String("flag", "value", "test flag")

	usage := Usage(fs)
	if !strings.Contains(usage, "Usage of db-catalyst:") {
		t.Fatalf("usage missing header: %q", usage)
	}
	if !strings.Contains(usage, "-flag") {
		t.Fatalf("usage missing flag definition: %q", usage)
	}
}
