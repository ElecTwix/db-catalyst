// Package logging provides a configured slog logger for db-catalyst.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// Options configures the default slog logger used by db-catalyst.
type Options struct {
	// Verbose toggles debug level logging when true.
	Verbose bool
	// Writer directs log output; defaults to os.Stderr when nil.
	Writer io.Writer
}

// New constructs a slog.Logger with db-catalyst defaults.
func New(opts Options) *slog.Logger {
	level := slog.LevelInfo
	if opts.Verbose {
		level = slog.LevelDebug
	}
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
