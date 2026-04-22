package cli

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Miss-you/go-symphony/internal/logging"
)

var logFileMu sync.Mutex

func DefaultLogFile(root string) string {
	if root == "" {
		if cwd, err := os.Getwd(); err == nil {
			root = cwd
		}
	}
	return filepath.Join(root, "log", "symphony.log")
}

func configureLogFile(root string) (func() error, string, error) {
	path := DefaultLogFile(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, path, err
	}

	logFileMu.Lock()
	previousLogger := slog.Default()
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()

	logger, lw, err := logging.New(logging.DefaultConfig(path))
	if err != nil {
		logFileMu.Unlock()
		return nil, path, err
	}

	slog.SetDefault(logger)
	log.SetOutput(lw)

	var once sync.Once
	restore := func() error {
		var closeErr error
		once.Do(func() {
			slog.SetDefault(previousLogger)
			log.SetOutput(previousWriter)
			log.SetFlags(previousFlags)
			log.SetPrefix(previousPrefix)
			closeErr = lw.Close()
			logFileMu.Unlock()
		})
		return closeErr
	}
	return restore, path, nil
}
