package logging

import (
	"bytes"
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
