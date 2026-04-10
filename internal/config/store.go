package config

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"
)

type StoreOption func(*storeOptions)

type storeOptions struct {
	path   string
	ticks  <-chan time.Time
	fileOps fileOps
	logf   func(string, ...any)
}

type fileOps struct {
	readFile func(string) ([]byte, error)
	statFile func(string) (fs.FileInfo, error)
	hash     func([]byte) uint64
}

type workflowStamp struct {
	modTime int64
	size    int64
	hash    uint64
}

type Store struct {
	mu         sync.RWMutex
	desiredPath string
	loadedPath  string
	workflow    Workflow
	stamp       workflowStamp
	fileOps     fileOps
	logf        func(string, ...any)
	ticks       <-chan time.Time
	stop        chan struct{}
	done        chan struct{}
	cleanup     func()
	closeOnce   sync.Once
}

func NewStore(opts ...StoreOption) (*Store, error) {
	options := storeOptions{
		fileOps: defaultFileOps(),
		logf:    log.Printf,
	}
	for _, opt := range opts {
		opt(&options)
	}

	desiredPath, err := resolvedWorkflowPath(options.path)
	if err != nil {
		return nil, err
	}

	workflow, stamp, err := loadState(desiredPath, options.fileOps)
	if err != nil {
		return nil, err
	}

	store := &Store{
		desiredPath: desiredPath,
		loadedPath:  desiredPath,
		workflow:    workflow,
		stamp:       stamp,
		fileOps:     options.fileOps,
		logf:        options.logf,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}

	if options.ticks != nil {
		store.ticks = options.ticks
	} else {
		ticker := time.NewTicker(time.Second)
		store.ticks = ticker.C
		store.cleanup = ticker.Stop
	}

	go store.run()

	return store, nil
}

func (s *Store) Current() (Workflow, error) {
	if err := s.attemptReload(false); err != nil {
		s.mu.RLock()
		workflow := s.workflow
		s.mu.RUnlock()
		return workflow, nil
	}

	s.mu.RLock()
	workflow := s.workflow
	s.mu.RUnlock()
	return workflow, nil
}

func (s *Store) ForceReload() error {
	return s.attemptReload(true)
}

func (s *Store) SetWorkflowPath(path string) error {
	resolved, err := resolvedWorkflowPath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.desiredPath = resolved
	s.mu.Unlock()

	return s.attemptReload(true)
}

func (s *Store) ClearWorkflowPath() error {
	resolved, err := resolvedWorkflowPath("")
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.desiredPath = resolved
	s.mu.Unlock()

	return s.attemptReload(true)
}

func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		close(s.stop)
		if s.cleanup != nil {
			s.cleanup()
		}
		<-s.done
	})
	return nil
}

func (s *Store) run() {
	defer close(s.done)

	for {
		select {
		case <-s.stop:
			return
		case <-s.ticks:
			_ = s.attemptReload(false)
		}
	}
}

func (s *Store) attemptReload(force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.reloadLocked(force)
}

func (s *Store) reloadLocked(force bool) error {
	shouldReload := force || s.desiredPath != s.loadedPath
	if !shouldReload {
		currentStamp, err := currentStamp(s.desiredPath, s.fileOps)
		if err == nil && currentStamp == s.stamp {
			return nil
		}
		shouldReload = true
	}

	if !shouldReload {
		return nil
	}

	workflow, stamp, err := loadState(s.desiredPath, s.fileOps)
	if err != nil {
		s.logf("failed to reload workflow path=%s err=%v; keeping last known good configuration", s.desiredPath, err)
		return err
	}

	s.workflow = workflow
	s.loadedPath = s.desiredPath
	s.stamp = stamp
	return nil
}

func loadState(path string, ops fileOps) (Workflow, workflowStamp, error) {
	content, err := ops.readFile(path)
	if err != nil {
		return Workflow{}, workflowStamp{}, &LoadError{Code: ErrMissingWorkflowFile, Path: path, Err: err}
	}

	stat, err := ops.statFile(path)
	if err != nil {
		return Workflow{}, workflowStamp{}, err
	}

	workflow, err := Parse(content, path)
	if err != nil {
		return Workflow{}, workflowStamp{}, err
	}

	return workflow, workflowStamp{
		modTime: stat.ModTime().UnixNano(),
		size:    stat.Size(),
		hash:    ops.hash(content),
	}, nil
}

func currentStamp(path string, ops fileOps) (workflowStamp, error) {
	content, err := ops.readFile(path)
	if err != nil {
		return workflowStamp{}, err
	}

	stat, err := ops.statFile(path)
	if err != nil {
		return workflowStamp{}, err
	}

	return workflowStamp{
		modTime: stat.ModTime().UnixNano(),
		size:    stat.Size(),
		hash:    ops.hash(content),
	}, nil
}

func defaultFileOps() fileOps {
	return fileOps{
		readFile: osReadFile,
		statFile: osStat,
		hash:     contentHash,
	}
}

func withWorkflowPath(path string) StoreOption {
	return func(opts *storeOptions) {
		opts.path = path
	}
}

func withTickChannel(ticks <-chan time.Time) StoreOption {
	return func(opts *storeOptions) {
		opts.ticks = ticks
	}
}

func withLogf(logf func(string, ...any)) StoreOption {
	return func(opts *storeOptions) {
		opts.logf = logf
	}
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func osStat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func contentHash(content []byte) uint64 {
	hasher := fnv.New64a()
	if _, err := hasher.Write(content); err != nil {
		panic(fmt.Sprintf("hash workflow content: %v", err))
	}
	return hasher.Sum64()
}
