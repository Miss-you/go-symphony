package cli

import (
	"log"
	"os"
	"path/filepath"
	"sync"
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
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, path, err
	}

	logFileMu.Lock()
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(file)

	var once sync.Once
	restore := func() error {
		var closeErr error
		once.Do(func() {
			log.SetOutput(previousWriter)
			log.SetFlags(previousFlags)
			log.SetPrefix(previousPrefix)
			closeErr = file.Close()
			logFileMu.Unlock()
		})
		return closeErr
	}
	return restore, path, nil
}
