package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewStoreFailsWithoutKnownGoodWorkflow(t *testing.T) {
	t.Parallel()

	_, err := NewStore(withWorkflowPath("missing/WORKFLOW.md"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStoreCurrentReloadsAfterTick(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "WORKFLOW.md", "First prompt\n")
	ticks := make(chan time.Time, 1)

	store, err := NewStore(
		withWorkflowPath(path),
		withTickChannel(ticks),
		withLogf(func(string, ...any) {}),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := overwriteFile(path, "Second prompt\n"); err != nil {
		t.Fatalf("overwrite workflow: %v", err)
	}

	ticks <- time.Now()

	got := waitForPrompt(t, store, "Second prompt")
	if got.PromptTemplate != "Second prompt" {
		t.Fatalf("unexpected prompt: %#v", got)
	}
}

func TestStoreForceReloadKeepsLastKnownGoodOnInvalidContent(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "WORKFLOW.md", "Initial prompt\n")
	var logs []string

	store, err := NewStore(
		withWorkflowPath(path),
		withLogf(func(format string, args ...any) {
			logs = append(logs, format)
		}),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := overwriteFile(path, "---\ntracker: [\n---\nBroken prompt\n"); err != nil {
		t.Fatalf("overwrite workflow: %v", err)
	}

	if err := store.ForceReload(); err == nil {
		t.Fatal("expected reload error")
	}

	got, err := store.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got.PromptTemplate != "Initial prompt" {
		t.Fatalf("expected last known good prompt, got %q", got.PromptTemplate)
	}
	if len(logs) == 0 {
		t.Fatal("expected reload failure log")
	}
}

func TestStoreSetWorkflowPathRetriesFailedPathSwitch(t *testing.T) {
	t.Parallel()

	initialPath := writeWorkflowFile(t, "WORKFLOW.md", "Initial prompt\n")
	nextPath := initialPath + ".next"

	store, err := NewStore(
		withWorkflowPath(initialPath),
		withLogf(func(string, ...any) {}),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.SetWorkflowPath(nextPath); err == nil {
		t.Fatal("expected path-switch error")
	}

	got, err := store.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got.PromptTemplate != "Initial prompt" {
		t.Fatalf("expected cached prompt, got %q", got.PromptTemplate)
	}

	if err := overwriteFile(nextPath, "Recovered prompt\n"); err != nil {
		t.Fatalf("write recovered workflow: %v", err)
	}

	got = waitForPrompt(t, store, "Recovered prompt")
	if got.PromptTemplate != "Recovered prompt" {
		t.Fatalf("expected recovered prompt, got %q", got.PromptTemplate)
	}
}

func TestStoreClearWorkflowPathSwitchesBackToDefaultPath(t *testing.T) {
	tempDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	initialPath := filepath.Join(tempDir, "EXPLICIT_WORKFLOW.md")
	if err := overwriteFile(initialPath, "Explicit prompt\n"); err != nil {
		t.Fatalf("write explicit workflow: %v", err)
	}
	defaultPath := filepath.Join(tempDir, workflowFileName)
	if err := overwriteFile(defaultPath, "Default prompt\n"); err != nil {
		t.Fatalf("write default workflow: %v", err)
	}
	actualWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd after chdir: %v", err)
	}

	store, err := NewStore(
		withWorkflowPath(initialPath),
		withLogf(func(string, ...any) {}),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.ClearWorkflowPath(); err != nil {
		t.Fatalf("ClearWorkflowPath: %v", err)
	}

	got := waitForPrompt(t, store, "Default prompt")
	expectedPath := filepath.Join(actualWD, workflowFileName)
	if got.Path != expectedPath {
		t.Fatalf("expected default path %q, got %q", expectedPath, got.Path)
	}
}

func waitForPrompt(t *testing.T, store *Store, want string) Workflow {
	t.Helper()

	for range 50 {
		got, err := store.Current()
		if err == nil && got.PromptTemplate == want {
			return got
		}
	}

	got, err := store.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	t.Fatalf("prompt mismatch: got %q want %q", got.PromptTemplate, want)
	return Workflow{}
}

func overwriteFile(path, content string) error {
	return writeFile(path, strings.NewReader(content))
}
