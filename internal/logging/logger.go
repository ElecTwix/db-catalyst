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

// Logger is a generic logging interface that abstracts slog.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// SlogAdapter adapts *slog.Logger to the Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new SlogAdapter.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: logger}
}

// Debug logs a debug message.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info logs an info message.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error logs an error message.
func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// With returns a new Logger with the given attributes.
func (s *SlogAdapter) With(args ...any) Logger {
	return &SlogAdapter{logger: s.logger.With(args...)}
}

// Ensure SlogAdapter implements Logger interface
var _ Logger = (*SlogAdapter)(nil)

// NopLogger is a logger that discards all output.
type NopLogger struct{}

// NewNopLogger creates a new NopLogger.
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// Debug is a no-op.
func (n *NopLogger) Debug(_ string, _ ...any) {}

// Info is a no-op.
func (n *NopLogger) Info(_ string, _ ...any) {}

// Warn is a no-op.
func (n *NopLogger) Warn(_ string, _ ...any) {}

// Error is a no-op.
func (n *NopLogger) Error(_ string, _ ...any) {}

// With returns the same NopLogger.
func (n *NopLogger) With(_ ...any) Logger {
	return n
}

// Ensure NopLogger implements Logger interface
var _ Logger = (*NopLogger)(nil)
