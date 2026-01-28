package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_DefaultLevel(t *testing.T) {
	t.Helper()
	var buf bytes.Buffer
	logger := New(Options{Writer: &buf})

	logger.Debug("debug suppressed")
	if got := buf.Len(); got != 0 {
		t.Fatalf("expected debug output to be suppressed, got %d bytes", got)
	}

	logger.Info("visible message")
	if out := buf.String(); !strings.Contains(out, "visible message") {
		t.Fatalf("expected info log to contain message, got %q", out)
	}
}

func TestNew_VerboseEnablesDebug(t *testing.T) {
	t.Helper()
	var buf bytes.Buffer
	logger := New(Options{Verbose: true, Writer: &buf})

	logger.Debug("debug visible")
	if out := buf.String(); !strings.Contains(out, "debug visible") {
		t.Fatalf("expected debug output when verbose, got %q", out)
	}
}

func TestSlogAdapter(t *testing.T) {
	t.Run("debug log", func(t *testing.T) {
		var buf bytes.Buffer
		slogLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logger := NewSlogAdapter(slogLogger)

		logger.Debug("debug message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, "debug message") {
			t.Errorf("output = %q, want to contain 'debug message'", output)
		}
		if !strings.Contains(output, "key=value") {
			t.Errorf("output = %q, want to contain 'key=value'", output)
		}
	})

	t.Run("info log", func(t *testing.T) {
		var buf bytes.Buffer
		slogLogger := slog.New(slog.NewTextHandler(&buf, nil))
		logger := NewSlogAdapter(slogLogger)

		logger.Info("info message", "count", 42)

		output := buf.String()
		if !strings.Contains(output, "info message") {
			t.Errorf("output = %q, want to contain 'info message'", output)
		}
	})

	t.Run("warn log", func(t *testing.T) {
		var buf bytes.Buffer
		slogLogger := slog.New(slog.NewTextHandler(&buf, nil))
		logger := NewSlogAdapter(slogLogger)

		logger.Warn("warn message")

		output := buf.String()
		if !strings.Contains(output, "warn message") {
			t.Errorf("output = %q, want to contain 'warn message'", output)
		}
	})

	t.Run("error log", func(t *testing.T) {
		var buf bytes.Buffer
		slogLogger := slog.New(slog.NewTextHandler(&buf, nil))
		logger := NewSlogAdapter(slogLogger)

		logger.Error("error message", "err", "something failed")

		output := buf.String()
		if !strings.Contains(output, "error message") {
			t.Errorf("output = %q, want to contain 'error message'", output)
		}
	})

	t.Run("with attributes", func(t *testing.T) {
		var buf bytes.Buffer
		slogLogger := slog.New(slog.NewTextHandler(&buf, nil))
		logger := NewSlogAdapter(slogLogger)

		child := logger.With("component", "test")
		child.Info("message")

		output := buf.String()
		if !strings.Contains(output, "component=test") {
			t.Errorf("output = %q, want to contain 'component=test'", output)
		}
	})
}

func TestNopLogger(t *testing.T) {
	logger := NewNopLogger()

	// These should not panic or produce output
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	child := logger.With("key", "value")
	child.Info("child message")

	// If we get here without panic, test passes
}

func TestLoggerInterface(t *testing.T) {
	// Verify both implementations satisfy the interface
	var _ Logger = (*SlogAdapter)(nil)
	var _ Logger = (*NopLogger)(nil)
}
