package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigValues(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("/tmp/example.log")

	if cfg.Path != "/tmp/example.log" {
		t.Fatalf("Path mismatch: got %q", cfg.Path)
	}
	if cfg.Level != slog.LevelDebug {
		t.Fatalf("Level mismatch: got %v want Debug", cfg.Level)
	}
	if cfg.MaxSize != 100 {
		t.Fatalf("MaxSize mismatch: got %d want 100", cfg.MaxSize)
	}
	if cfg.MaxBackups != 5 {
		t.Fatalf("MaxBackups mismatch: got %d want 5", cfg.MaxBackups)
	}
	if cfg.MaxAge != 30 {
		t.Fatalf("MaxAge mismatch: got %d want 30", cfg.MaxAge)
	}
	if cfg.JSON {
		t.Fatalf("JSON default should be false")
	}
}

func TestNewCreatesParentDirectoryAndWritesText(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	logPath := filepath.Join(root, "nested", "deep", "symphony.log")

	logger, lw, err := New(DefaultConfig(logPath))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := lw.Close(); err != nil {
			t.Errorf("close lumberjack: %v", err)
		}
	})

	logger.Info("hello", "key", "value")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "hello") {
		t.Fatalf("expected message in log, got %q", body)
	}
	if !strings.Contains(body, "key=value") {
		t.Fatalf("expected text-format key=value attr, got %q", body)
	}
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("expected text handler output, got JSON-looking body: %q", body)
	}
}

func TestNewJSONHandlerEmitsJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	logPath := filepath.Join(root, "symphony.log")

	cfg := DefaultConfig(logPath)
	cfg.JSON = true

	logger, lw, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = lw.Close() })

	logger.Info("hello", "key", "value")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		t.Fatalf("expected single JSON object line, got %q", line)
	}
	if !strings.Contains(line, `"msg":"hello"`) {
		t.Fatalf("expected JSON msg field, got %q", line)
	}
	if !strings.Contains(line, `"key":"value"`) {
		t.Fatalf("expected JSON key field, got %q", line)
	}
}

func TestNewLevelFilterDropsBelowThreshold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	logPath := filepath.Join(root, "symphony.log")

	cfg := DefaultConfig(logPath)
	cfg.Level = slog.LevelInfo

	logger, lw, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = lw.Close() })

	logger.Debug("debug-should-be-dropped")
	logger.Info("info-should-survive")
	logger.Warn("warn-should-survive")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	body := string(data)
	if strings.Contains(body, "debug-should-be-dropped") {
		t.Fatalf("debug record leaked through Info-level filter: %q", body)
	}
	if !strings.Contains(body, "info-should-survive") {
		t.Fatalf("info record missing: %q", body)
	}
	if !strings.Contains(body, "warn-should-survive") {
		t.Fatalf("warn record missing: %q", body)
	}
}

func TestNewReturnsErrorWhenParentPathIsFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}

	logPath := filepath.Join(blocker, "symphony.log")

	logger, lw, err := New(DefaultConfig(logPath))
	if err == nil {
		_ = lw.Close()
		t.Fatalf("expected error when parent path is a regular file, got logger=%v", logger)
	}
	if !strings.Contains(err.Error(), "create log directory") {
		t.Fatalf("expected wrapped 'create log directory' error, got %v", err)
	}
}

func TestNewRotatesWhenMaxSizeExceeded(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	logPath := filepath.Join(root, "symphony.log")

	cfg := DefaultConfig(logPath)
	cfg.MaxSize = 1 // 1 megabyte rotation threshold
	cfg.JSON = true

	logger, lw, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = lw.Close() })

	// Each Info call writes at least a few hundred bytes once attrs are JSON-encoded.
	// Push well past 1 MiB to guarantee at least one rotation event.
	payload := strings.Repeat("x", 1024)
	for i := 0; i < 2000; i++ {
		logger.Info("rot", "i", i, "blob", payload)
	}

	if err := lw.Close(); err != nil {
		t.Fatalf("close lumberjack: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}

	var (
		sawCurrent bool
		backups    int
	)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		switch {
		case name == "symphony.log":
			sawCurrent = true
		case strings.HasPrefix(name, "symphony-") && (strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz")):
			backups++
		}
	}

	if !sawCurrent {
		t.Fatalf("expected current log file %q to exist after rotation", logPath)
	}
	if backups == 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected at least one rotated backup file, got entries: %v", names)
	}
}
