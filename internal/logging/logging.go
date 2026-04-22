// Package logging provides structured logging initialization with rotation.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger initialization parameters.
type Config struct {
	Path       string
	Level      slog.Level
	MaxSize    int  // megabytes
	MaxBackups int
	MaxAge     int  // days
	JSON       bool
}

// DefaultConfig returns a sensible default configuration for the given path.
func DefaultConfig(path string) Config {
	return Config{
		Path:       path,
		Level:      slog.LevelDebug,
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		JSON:       false,
	}
}

// New creates a structured slog.Logger with lumberjack-based file rotation.
// The returned *lumberjack.Logger should be closed on application shutdown.
func New(cfg Config) (*slog.Logger, *lumberjack.Logger, error) {
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	lw := &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true,
		LocalTime:  true,
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.Level == slog.LevelDebug,
	}

	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(lw, opts)
	} else {
		handler = slog.NewTextHandler(lw, opts)
	}

	return slog.New(handler), lw, nil
}
