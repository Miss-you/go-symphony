package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Miss-you/go-symphony/internal/config"
)

func TestPathForIdentifierNormalizesUnsafeAndFallsBackToIssue(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks:     config.HookSettings{TimeoutMS: 100},
	})

	firstPath, err := manager.PathForIdentifier("MT/123:alpha", "")
	if err != nil {
		t.Fatalf("PathForIdentifier returned error: %v", err)
	}
	secondPath, err := manager.PathForIdentifier("MT/123:alpha", "")
	if err != nil {
		t.Fatalf("PathForIdentifier returned error on second call: %v", err)
	}
	fallbackPath, err := manager.PathForIdentifier("", "")
	if err != nil {
		t.Fatalf("PathForIdentifier fallback returned error: %v", err)
	}

	if filepath.Base(firstPath) != "MT_123_alpha" {
		t.Fatalf("normalized basename = %q, want %q", filepath.Base(firstPath), "MT_123_alpha")
	}
	if firstPath != secondPath {
		t.Fatalf("deterministic path mismatch: %q vs %q", firstPath, secondPath)
	}
	if filepath.Base(fallbackPath) != "issue" {
		t.Fatalf("fallback basename = %q, want %q", filepath.Base(fallbackPath), "issue")
	}
}

func TestPathForIdentifierCanonicalizesSymlinkedRoot(t *testing.T) {
	t.Parallel()

	testRoot := t.TempDir()
	actualRoot := filepath.Join(testRoot, "actual")
	linkedRoot := filepath.Join(testRoot, "linked")
	if err := os.MkdirAll(actualRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(actualRoot): %v", err)
	}
	if err := os.Symlink(actualRoot, linkedRoot); err != nil {
		t.Fatalf("Symlink(linkedRoot): %v", err)
	}

	manager := NewManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: linkedRoot},
		Hooks:     config.HookSettings{TimeoutMS: 100},
	})

	got, err := manager.PathForIdentifier("MT-LINK", "")
	if err != nil {
		t.Fatalf("PathForIdentifier returned error: %v", err)
	}

	canonicalRoot, err := canonicalizePath(actualRoot)
	if err != nil {
		t.Fatalf("canonicalizePath(actualRoot): %v", err)
	}
	want := filepath.Join(canonicalRoot, "MT-LINK")
	if got != want {
		t.Fatalf("canonical workspace path = %q, want %q", got, want)
	}
}

func TestPathForIdentifierRejectsRootEquality(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks:     config.HookSettings{TimeoutMS: 100},
	})

	_, err := manager.PathForIdentifier(".", "")
	if err == nil {
		t.Fatal("PathForIdentifier returned nil error, want root collision error")
	}

	var workspaceErr *Error
	if !errors.As(err, &workspaceErr) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if workspaceErr.Kind != ErrWorkspaceEqualsRoot {
		t.Fatalf("error kind = %q, want %q", workspaceErr.Kind, ErrWorkspaceEqualsRoot)
	}
}

func TestPathForIdentifierRejectsOutsideRootAndSymlinkEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	escapeLink := filepath.Join(root, "MT-SYM")
	if err := os.Symlink(outside, escapeLink); err != nil {
		t.Fatalf("Symlink(escapeLink): %v", err)
	}

	manager := NewManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks:     config.HookSettings{TimeoutMS: 100},
	})

	if _, err := manager.PathForIdentifier("..", ""); err == nil {
		t.Fatal("PathForIdentifier(\"..\") returned nil error, want outside-root rejection")
	} else {
		var workspaceErr *Error
		if !errors.As(err, &workspaceErr) {
			t.Fatalf("outside-root error type = %T, want *Error", err)
		}
		if workspaceErr.Kind != ErrWorkspaceOutsideRoot {
			t.Fatalf("outside-root error kind = %q, want %q", workspaceErr.Kind, ErrWorkspaceOutsideRoot)
		}
	}

	if _, err := manager.PathForIdentifier("MT-SYM", ""); err == nil {
		t.Fatal("PathForIdentifier(\"MT-SYM\") returned nil error, want symlink-escape rejection")
	} else {
		var workspaceErr *Error
		if !errors.As(err, &workspaceErr) {
			t.Fatalf("symlink-escape error type = %T, want *Error", err)
		}
		if workspaceErr.Kind != ErrWorkspaceSymlinkEscape {
			t.Fatalf("symlink-escape error kind = %q, want %q", workspaceErr.Kind, ErrWorkspaceSymlinkEscape)
		}
	}
}
