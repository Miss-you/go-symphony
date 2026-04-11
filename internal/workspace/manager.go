package workspace

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/runner"
)

type ErrorKind string

const (
	ErrWorkspacePathUnreadable ErrorKind = "workspace_path_unreadable"
	ErrWorkspaceEqualsRoot     ErrorKind = "workspace_equals_root"
	ErrWorkspaceOutsideRoot    ErrorKind = "workspace_outside_root"
	ErrWorkspaceSymlinkEscape  ErrorKind = "workspace_symlink_escape"
	ErrWorkspaceHookTimeout    ErrorKind = "workspace_hook_timeout"
	ErrWorkspaceHookFailed     ErrorKind = "workspace_hook_failed"
	ErrWorkspacePrepareFailed  ErrorKind = "workspace_prepare_failed"
	ErrWorkspaceRemoveFailed   ErrorKind = "workspace_remove_failed"
)

type Error struct {
	Kind       ErrorKind
	Path       string
	Root       string
	Hook       string
	WorkerHost string
	Status     int
	Output     string
	Timeout    time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	parts := []string{string(e.Kind)}
	if e.Hook != "" {
		parts = append(parts, "hook="+e.Hook)
	}
	if e.Path != "" {
		parts = append(parts, "path="+e.Path)
	}
	if e.Root != "" {
		parts = append(parts, "root="+e.Root)
	}
	if e.WorkerHost != "" {
		parts = append(parts, "worker_host="+e.WorkerHost)
	}
	if e.Status != 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.Status))
	}
	if e.Timeout > 0 {
		parts = append(parts, "timeout="+e.Timeout.String())
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Manager struct {
	root        string
	hooks       config.HookSettings
	workerHosts []string
	transport   transport
	logger      *log.Logger
}

type CreateResult struct {
	Path    string
	Created bool
}

type commandResult struct {
	output string
	status int
}

type transport interface {
	EnsureWorkspace(ctx context.Context, workerHost, path string) (string, bool, error)
	RunCommand(ctx context.Context, workerHost, dir, command string, timeout time.Duration) (commandResult, error)
	RemoveWorkspace(ctx context.Context, workerHost, path string) error
}

type runnerTransport struct {
	executor runner.Executor
	timeout  time.Duration
}

type hookPolicy struct {
	name       string
	command    string
	bestEffort bool
}

func NewManager(settings config.Settings) *Manager {
	hookTimeout := time.Duration(settings.Hooks.TimeoutMS) * time.Millisecond
	return &Manager{
		root:        settings.Workspace.Root,
		hooks:       settings.Hooks,
		workerHosts: normalizeWorkerHosts(settings.Worker.SSHHosts),
		transport: runnerTransport{
			executor: runner.NewExecutor(),
			timeout:  hookTimeout,
		},
		logger: log.Default(),
	}
}

func (m *Manager) PathForIdentifier(identifier, workerHost string) (string, error) {
	safeID := safeIdentifier(identifier)
	workspacePath := filepath.Join(m.root, safeID)
	if workerHost != "" {
		return validateRemotePath(workspacePath, workerHost)
	}
	return validateLocalPath(m.root, workspacePath)
}

func (m *Manager) Create(identifier, workerHost string) (CreateResult, error) {
	workspacePath, err := m.PathForIdentifier(identifier, workerHost)
	if err != nil {
		return CreateResult{}, err
	}
	resolvedPath, created, err := m.transport.EnsureWorkspace(context.Background(), workerHost, workspacePath)
	if err != nil {
		return CreateResult{}, &Error{Kind: ErrWorkspacePrepareFailed, Path: workspacePath, WorkerHost: workerHost, Err: err}
	}
	if created {
		if err := m.runHook(resolvedPath, workerHost, hookPolicy{
			name:    "after_create",
			command: m.hooks.AfterCreate,
		}); err != nil {
			return CreateResult{}, err
		}
	}
	return CreateResult{Path: resolvedPath, Created: created}, nil
}

func (m *Manager) RunWithHooks(workspacePath, _ string, workerHost string, run func() error) (runErr error) {
	defer func() {
		_ = m.runHook(workspacePath, workerHost, hookPolicy{
			name:       "after_run",
			command:    m.hooks.AfterRun,
			bestEffort: true,
		})
	}()

	if err := m.runHook(workspacePath, workerHost, hookPolicy{
		name:    "before_run",
		command: m.hooks.BeforeRun,
	}); err != nil {
		return err
	}
	if run == nil {
		return nil
	}
	return run()
}

func (m *Manager) Remove(workspacePath, _ string, workerHost string) error {
	if workerHost == "" {
		validatedPath, err := validateLocalPath(m.root, workspacePath)
		if err != nil {
			return err
		}
		workspacePath = validatedPath
		if info, err := os.Stat(workspacePath); err == nil {
			if info.IsDir() {
				_ = m.runHook(workspacePath, workerHost, hookPolicy{
					name:       "before_remove",
					command:    m.hooks.BeforeRemove,
					bestEffort: true,
				})
			}
		} else if errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			return &Error{Kind: ErrWorkspaceRemoveFailed, Path: workspacePath, Err: err}
		}
		if err := m.transport.RemoveWorkspace(context.Background(), workerHost, workspacePath); err != nil {
			return &Error{Kind: ErrWorkspaceRemoveFailed, Path: workspacePath, Err: err}
		}
		return nil
	}

	validatedPath, err := validateRemotePath(workspacePath, workerHost)
	if err != nil {
		return err
	}
	workspacePath = validatedPath
	_ = m.runHook(workspacePath, workerHost, hookPolicy{
		name:       "before_remove",
		command:    m.hooks.BeforeRemove,
		bestEffort: true,
	})
	if err := m.transport.RemoveWorkspace(context.Background(), workerHost, workspacePath); err != nil {
		return &Error{Kind: ErrWorkspaceRemoveFailed, Path: workspacePath, WorkerHost: workerHost, Err: err}
	}
	return nil
}

func (m *Manager) RemoveIssueWorkspaces(identifier, workerHost string) error {
	if workerHost != "" {
		workspacePath, err := m.PathForIdentifier(identifier, workerHost)
		if err != nil {
			return err
		}
		return m.Remove(workspacePath, identifier, workerHost)
	}
	if len(m.workerHosts) > 0 {
		var errs []error
		for _, host := range m.workerHosts {
			if err := m.RemoveIssueWorkspaces(identifier, host); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	workspacePath, err := m.PathForIdentifier(identifier, "")
	if err != nil {
		return err
	}
	return m.Remove(workspacePath, identifier, "")
}

func safeIdentifier(identifier string) string {
	if strings.TrimSpace(identifier) == "" {
		return "issue"
	}

	var builder strings.Builder
	builder.Grow(len(identifier))
	for _, r := range identifier {
		switch {
		case r == '.' || r == '_' || r == '-':
			builder.WriteRune(r)
		case 'a' <= r && r <= 'z':
			builder.WriteRune(r)
		case 'A' <= r && r <= 'Z':
			builder.WriteRune(r)
		case '0' <= r && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	if builder.Len() == 0 {
		return "issue"
	}
	return builder.String()
}

func validateLocalPath(root, workspacePath string) (string, error) {
	canonicalWorkspace, err := canonicalizePath(workspacePath)
	if err != nil {
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, Err: err}
	}
	canonicalRoot, err := canonicalizePath(root)
	if err != nil {
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: root, Err: err}
	}
	if canonicalWorkspace == canonicalRoot {
		return "", &Error{Kind: ErrWorkspaceEqualsRoot, Path: canonicalWorkspace, Root: canonicalRoot}
	}
	if isWithinRoot(canonicalWorkspace, canonicalRoot) {
		return canonicalWorkspace, nil
	}

	expandedWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, Err: err}
	}
	expandedRoot, err := filepath.Abs(root)
	if err != nil {
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: root, Err: err}
	}
	if isWithinRoot(filepath.Clean(expandedWorkspace), filepath.Clean(expandedRoot)) {
		return "", &Error{Kind: ErrWorkspaceSymlinkEscape, Path: filepath.Clean(expandedWorkspace), Root: canonicalRoot}
	}
	return "", &Error{Kind: ErrWorkspaceOutsideRoot, Path: canonicalWorkspace, Root: canonicalRoot}
}

func validateRemotePath(workspacePath, workerHost string) (string, error) {
	switch {
	case strings.TrimSpace(workspacePath) == "":
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, WorkerHost: workerHost, Err: errors.New("empty path")}
	case strings.Contains(workspacePath, "\n"), strings.Contains(workspacePath, "\r"), strings.ContainsRune(workspacePath, '\x00'):
		return "", &Error{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, WorkerHost: workerHost, Err: errors.New("invalid characters")}
	default:
		return workspacePath, nil
	}
}

func (m *Manager) runHook(workspacePath, workerHost string, policy hookPolicy) error {
	if strings.TrimSpace(policy.command) == "" {
		return nil
	}
	timeout := time.Duration(m.hooks.TimeoutMS) * time.Millisecond
	result, err := m.transport.RunCommand(context.Background(), workerHost, workspacePath, policy.command, timeout)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			hookErr := &Error{
				Kind:       ErrWorkspaceHookTimeout,
				Hook:       policy.name,
				Path:       workspacePath,
				WorkerHost: workerHost,
				Timeout:    timeout,
				Err:        err,
			}
			if policy.bestEffort {
				m.logBestEffortHookError(hookErr)
				return nil
			}
			return hookErr
		}
		if policy.bestEffort {
			m.logBestEffortHookError(&Error{
				Kind:       ErrWorkspaceHookFailed,
				Hook:       policy.name,
				Path:       workspacePath,
				WorkerHost: workerHost,
				Err:        err,
			})
			return nil
		}
		return &Error{
			Kind:       ErrWorkspaceHookFailed,
			Hook:       policy.name,
			Path:       workspacePath,
			WorkerHost: workerHost,
			Err:        err,
		}
	}
	if result.status != 0 {
		hookErr := &Error{
			Kind:       ErrWorkspaceHookFailed,
			Hook:       policy.name,
			Path:       workspacePath,
			WorkerHost: workerHost,
			Status:     result.status,
			Output:     result.output,
		}
		if policy.bestEffort {
			m.logBestEffortHookError(hookErr)
			return nil
		}
		return hookErr
	}
	return nil
}

func canonicalizePath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolutePath = filepath.Clean(absolutePath)

	currentRoot, segments := splitAbsolutePath(absolutePath)
	current := currentRoot
	for index, segment := range segments {
		candidate := filepath.Join(current, segment)
		info, err := os.Lstat(candidate)
		switch {
		case err == nil && info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(candidate)
			if err != nil {
				return "", err
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(current, target)
			}
			remaining := filepath.Join(segments[index+1:]...)
			if remaining != "" {
				target = filepath.Join(target, remaining)
			}
			return canonicalizePath(target)
		case err == nil:
			current = candidate
		case errors.Is(err, os.ErrNotExist):
			remaining := filepath.Join(segments[index:]...)
			if remaining == "" {
				return current, nil
			}
			return filepath.Join(current, remaining), nil
		default:
			return "", err
		}
	}
	return current, nil
}

func splitAbsolutePath(path string) (string, []string) {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	root := volume + string(os.PathSeparator)
	remainder := strings.TrimPrefix(clean, root)
	if root == string(os.PathSeparator) {
		remainder = strings.TrimPrefix(clean, string(os.PathSeparator))
	}
	if remainder == "" {
		return root, nil
	}
	return root, strings.Split(remainder, string(os.PathSeparator))
}

func isWithinRoot(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func normalizeWorkerHosts(hosts []string) []string {
	normalized := make([]string, 0, len(hosts))
	for _, host := range hosts {
		trimmed := strings.TrimSpace(host)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func (m *Manager) logBestEffortHookError(err error) {
	if m == nil || m.logger == nil || err == nil {
		return
	}
	m.logger.Printf("workspace hook ignored: %v", err)
}

func (t runnerTransport) EnsureWorkspace(ctx context.Context, workerHost, path string) (string, bool, error) {
	if workerHost != "" {
		result, err := t.run(ctx, runner.CommandRequest{
			Host:    workerHost,
			Command: remoteEnsureWorkspaceCommand(path),
			Timeout: t.timeout,
		})
		if err != nil {
			return "", false, err
		}
		if result.status != 0 {
			return "", false, fmt.Errorf("remote workspace prepare failed with status %d: %s", result.status, result.output)
		}
		return parseRemoteEnsureWorkspaceOutput(result.output)
	}
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return path, false, nil
		}
		if err := os.RemoveAll(path); err != nil {
			return "", false, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", false, err
	}
	return path, true, nil
}

func (t runnerTransport) RunCommand(ctx context.Context, workerHost, dir, command string, timeout time.Duration) (commandResult, error) {
	return t.run(ctx, runner.CommandRequest{
		Host:    workerHost,
		Dir:     dir,
		Command: command,
		Timeout: timeout,
	})
}

func (t runnerTransport) RemoveWorkspace(ctx context.Context, workerHost, path string) error {
	if workerHost != "" {
		result, err := t.run(ctx, runner.CommandRequest{
			Host:    workerHost,
			Command: "rm -rf -- " + quoteShell(path),
			Timeout: t.timeout,
		})
		if err != nil {
			return err
		}
		if result.status != 0 {
			return fmt.Errorf("remote workspace remove failed with status %d: %s", result.status, result.output)
		}
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return nil
}

func (t runnerTransport) run(ctx context.Context, req runner.CommandRequest) (commandResult, error) {
	executor := t.executor
	if executor == nil {
		executor = runner.NewExecutor()
	}
	result, err := executor.RunCommand(ctx, req)
	if err != nil {
		return commandResult{}, err
	}
	return commandResult{output: result.Output, status: result.Status}, nil
}

const remoteWorkspaceMarker = "__go_symphony_workspace__:"

func remoteEnsureWorkspaceCommand(path string) string {
	quotedPath := quoteShell(path)
	return strings.Join([]string{
		"set -eu",
		"workspace=" + quotedPath,
		"created=0",
		`if [ -e "$workspace" ] && [ ! -d "$workspace" ]; then rm -rf -- "$workspace"; fi`,
		`if [ ! -d "$workspace" ]; then mkdir -p -- "$workspace"; created=1; fi`,
		`cd "$workspace"`,
		`printf '` + remoteWorkspaceMarker + `%s:%s\n' "$created" "$(pwd -P)"`,
	}, "\n")
}

func parseRemoteEnsureWorkspaceOutput(output string) (string, bool, error) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, remoteWorkspaceMarker) {
			continue
		}
		payload := strings.TrimPrefix(line, remoteWorkspaceMarker)
		parts := strings.SplitN(payload, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			break
		}
		switch parts[0] {
		case "0":
			return parts[1], false, nil
		case "1":
			return parts[1], true, nil
		}
		break
	}
	return "", false, errors.New("remote workspace marker not found")
}

func quoteShell(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, unsafeShellRune) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func unsafeShellRune(r rune) bool {
	return !safeShellRune(r)
}

func safeShellRune(r rune) bool {
	return r == '/' || r == '.' || r == '_' || r == '-' || ('0' <= r && r <= '9') || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}
