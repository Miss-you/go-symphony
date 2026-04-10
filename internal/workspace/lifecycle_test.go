package workspace

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
)

func TestCreateReusesExistingDirectoryWithoutRunningAfterCreate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			AfterCreate: "echo after-create",
			TimeoutMS:   100,
		},
	})

	path := filepath.Join(root, "MT-REUSE")
	fake.ensurePath = path
	fake.ensureCreated = false

	result, err := manager.Create("MT-REUSE", "")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if result.Created {
		t.Fatal("Create marked existing directory as created")
	}
	if result.Path != path {
		t.Fatalf("Create path = %q, want %q", result.Path, path)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("after_create ran for reused workspace: %+v", fake.commands)
	}
}

func TestCreateReplacesStaleFilePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	stalePath := filepath.Join(root, "MT-STALE")
	if err := os.WriteFile(stalePath, []byte("old state"), 0o644); err != nil {
		t.Fatalf("WriteFile(stalePath): %v", err)
	}

	manager := NewManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks:     config.HookSettings{TimeoutMS: 100},
	})

	result, err := manager.Create("MT-STALE", "")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if !result.Created {
		t.Fatal("Create did not mark stale path replacement as created")
	}
	if info, err := os.Stat(result.Path); err != nil {
		t.Fatalf("Stat(result.Path): %v", err)
	} else if !info.IsDir() {
		t.Fatalf("result path is not a directory: mode=%v", info.Mode())
	}
}

func TestRunWithHooksBeforeRunFailureStillRunsAfterRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRun: "echo before-run",
			AfterRun:  "echo after-run",
			TimeoutMS: 100,
		},
	})

	fake.commandResults = []fakeCommandResult{
		{status: 17, output: "nope"},
		{status: 0, output: "after"},
	}

	ranBody := false
	err := manager.RunWithHooks(filepath.Join(root, "MT-HOOKS"), "MT-HOOKS", "", func() error {
		ranBody = true
		return nil
	})
	if err == nil {
		t.Fatal("RunWithHooks returned nil error, want before_run failure")
	}
	if ranBody {
		t.Fatal("run body executed despite before_run failure")
	}
	if len(fake.commands) != 2 {
		t.Fatalf("hook calls = %d, want 2", len(fake.commands))
	}
	if fake.commands[0].command != "echo before-run" || fake.commands[1].command != "echo after-run" {
		t.Fatalf("hook order = %+v, want before_run then after_run", fake.commands)
	}
}

func TestRunWithHooksBodyFailureStillRunsAfterRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRun: "echo before-run",
			AfterRun:  "echo after-run",
			TimeoutMS: 100,
		},
	})

	fake.commandResults = []fakeCommandResult{
		{status: 0, output: "before"},
		{status: 0, output: "after"},
	}

	bodyErr := errors.New("run failed")
	err := manager.RunWithHooks(filepath.Join(root, "MT-BODY"), "MT-BODY", "", func() error {
		return bodyErr
	})
	if !errors.Is(err, bodyErr) {
		t.Fatalf("RunWithHooks error = %v, want body error %v", err, bodyErr)
	}
	if len(fake.commands) != 2 {
		t.Fatalf("hook calls = %d, want 2", len(fake.commands))
	}
	if fake.commands[1].command != "echo after-run" {
		t.Fatalf("second hook command = %q, want %q", fake.commands[1].command, "echo after-run")
	}
}

func TestRemoveContinuesWhenBeforeRemoveFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRemove: "echo before-remove",
			TimeoutMS:    100,
		},
	})

	path := filepath.Join(root, "MT-REMOVE")
	fake.commandResults = []fakeCommandResult{{status: 17, output: "nope"}}

	if err := manager.Remove(path, "MT-REMOVE", "worker-a"); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if len(fake.commands) != 1 || fake.commands[0].command != "echo before-remove" {
		t.Fatalf("before_remove commands = %+v, want one best-effort call", fake.commands)
	}
	if len(fake.removals) != 1 || fake.removals[0].workerHost != "worker-a" {
		t.Fatalf("remove calls = %+v, want one host-addressed removal", fake.removals)
	}
}

func TestRemoveIssueWorkspacesFansOutAcrossConfiguredHosts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Worker: config.WorkerSettings{
			SSHHosts: []string{"worker-a", "worker-b"},
		},
		Hooks: config.HookSettings{TimeoutMS: 100},
	})

	if err := manager.RemoveIssueWorkspaces("MT-FANOUT", ""); err != nil {
		t.Fatalf("RemoveIssueWorkspaces returned error: %v", err)
	}

	gotHosts := []string{fake.removals[0].workerHost, fake.removals[1].workerHost}
	if !slices.Equal(gotHosts, []string{"worker-a", "worker-b"}) {
		t.Fatalf("fan-out hosts = %v, want [worker-a worker-b]", gotHosts)
	}
}

func TestRemoveIssueWorkspacesSkipsBlankWorkerHosts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Worker: config.WorkerSettings{
			SSHHosts: []string{" ", "worker-a", "", "worker-b"},
		},
		Hooks: config.HookSettings{TimeoutMS: 100},
	})

	if err := manager.RemoveIssueWorkspaces("MT-FANOUT", ""); err != nil {
		t.Fatalf("RemoveIssueWorkspaces returned error: %v", err)
	}

	gotHosts := []string{fake.removals[0].workerHost, fake.removals[1].workerHost}
	if !slices.Equal(gotHosts, []string{"worker-a", "worker-b"}) {
		t.Fatalf("fan-out hosts with blanks = %v, want [worker-a worker-b]", gotHosts)
	}
}

func TestCreateAfterCreateFailureIsFatalAndStructured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			AfterCreate: "echo after-create",
			TimeoutMS:   100,
		},
	})

	path := filepath.Join(root, "MT-CREATE-FAIL")
	fake.ensurePath = path
	fake.ensureCreated = true
	fake.commandResults = []fakeCommandResult{{status: 17, output: "nope"}}

	_, err := manager.Create("MT-CREATE-FAIL", "worker-a")
	if err == nil {
		t.Fatal("Create returned nil error, want after_create failure")
	}
	var workspaceErr *Error
	if !errors.As(err, &workspaceErr) {
		t.Fatalf("Create error type = %T, want *Error", err)
	}
	if workspaceErr.Kind != ErrWorkspaceHookFailed {
		t.Fatalf("Create error kind = %q, want %q", workspaceErr.Kind, ErrWorkspaceHookFailed)
	}
	if len(fake.commands) != 1 || fake.commands[0].workerHost != "worker-a" {
		t.Fatalf("after_create calls = %+v, want one host-addressed hook call", fake.commands)
	}
}

func TestRunWithHooksBeforeRunTimeoutIsFatalButAfterRunStillExecutes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRun: "echo before-run",
			AfterRun:  "echo after-run",
			TimeoutMS: 100,
		},
	})

	fake.commandResults = []fakeCommandResult{
		{err: context.DeadlineExceeded},
		{status: 0, output: "after"},
	}

	err := manager.RunWithHooks(filepath.Join(root, "MT-TIMEOUT"), "MT-TIMEOUT", "", func() error {
		t.Fatal("run body should not execute on before_run timeout")
		return nil
	})
	if err == nil {
		t.Fatal("RunWithHooks returned nil error, want before_run timeout")
	}
	var workspaceErr *Error
	if !errors.As(err, &workspaceErr) {
		t.Fatalf("RunWithHooks error type = %T, want *Error", err)
	}
	if workspaceErr.Kind != ErrWorkspaceHookTimeout {
		t.Fatalf("RunWithHooks error kind = %q, want %q", workspaceErr.Kind, ErrWorkspaceHookTimeout)
	}
	if len(fake.commands) != 2 || fake.commands[1].command != "echo after-run" {
		t.Fatalf("hook calls = %+v, want before_run timeout followed by after_run", fake.commands)
	}
}

func TestRunWithHooksIgnoresAfterRunTimeoutAndDoesNotImplicitlyCleanup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRun: "echo before-run",
			AfterRun:  "echo after-run",
			TimeoutMS: 100,
		},
	})

	fake.commandResults = []fakeCommandResult{
		{status: 0, output: "before"},
		{err: context.DeadlineExceeded},
	}

	if err := manager.RunWithHooks(filepath.Join(root, "MT-NO-CLEANUP"), "MT-NO-CLEANUP", "", func() error {
		return nil
	}); err != nil {
		t.Fatalf("RunWithHooks returned error despite best-effort after_run: %v", err)
	}
	if len(fake.removals) != 0 {
		t.Fatalf("implicit cleanup calls = %+v, want none", fake.removals)
	}
}

func TestRunWithHooksLogsIgnoredAfterRunFailure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, fake := newFakeManager(config.Settings{
		Workspace: config.WorkspaceSettings{Root: root},
		Hooks: config.HookSettings{
			BeforeRun: "echo before-run",
			AfterRun:  "echo after-run",
			TimeoutMS: 100,
		},
	})
	var logs bytes.Buffer
	manager.logger = log.New(&logs, "", 0)

	fake.commandResults = []fakeCommandResult{
		{status: 0, output: "before"},
		{status: 17, output: "after failed"},
	}

	if err := manager.RunWithHooks(filepath.Join(root, "MT-LOG"), "MT-LOG", "", func() error {
		return nil
	}); err != nil {
		t.Fatalf("RunWithHooks returned error: %v", err)
	}
	if logs.Len() == 0 {
		t.Fatal("best-effort after_run failure did not produce a log entry")
	}
	if !strings.Contains(logs.String(), "after_run") {
		t.Fatalf("log output = %q, want hook name", logs.String())
	}
}

type fakeManagerTransport struct {
	ensurePath    string
	ensureCreated bool
	ensureErr     error
	removeErr     error

	commandResults []fakeCommandResult
	commands       []fakeCommandCall
	removals       []fakeRemoveCall
}

type fakeCommandResult struct {
	output string
	status int
	err    error
}

type fakeCommandCall struct {
	workerHost string
	dir        string
	command    string
	timeout    time.Duration
}

type fakeRemoveCall struct {
	workerHost string
	path       string
}

func (f *fakeManagerTransport) EnsureWorkspace(_ context.Context, workerHost, path string) (string, bool, error) {
	if f.ensureErr != nil {
		return "", false, f.ensureErr
	}
	if f.ensurePath == "" {
		f.ensurePath = path
	}
	return f.ensurePath, f.ensureCreated, nil
}

func (f *fakeManagerTransport) RunCommand(_ context.Context, workerHost, dir, command string, timeout time.Duration) (commandResult, error) {
	f.commands = append(f.commands, fakeCommandCall{
		workerHost: workerHost,
		dir:        dir,
		command:    command,
		timeout:    timeout,
	})
	if len(f.commandResults) == 0 {
		return commandResult{}, nil
	}
	result := f.commandResults[0]
	f.commandResults = f.commandResults[1:]
	return commandResult{output: result.output, status: result.status}, result.err
}

func (f *fakeManagerTransport) RemoveWorkspace(_ context.Context, workerHost, path string) error {
	f.removals = append(f.removals, fakeRemoveCall{workerHost: workerHost, path: path})
	return f.removeErr
}

func newFakeManager(settings config.Settings) (*Manager, *fakeManagerTransport) {
	fake := &fakeManagerTransport{}
	manager := NewManager(settings)
	manager.transport = fake
	return manager, fake
}
