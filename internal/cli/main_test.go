package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/dashboard"
	"github.com/Miss-you/go-symphony/internal/domain"
)

const testAckFlag = "--i-understand-that-this-will-be-running-without-the-usual-guardrails"

func TestMainRequiresGuardrailsAcknowledgementBeforeSideEffects(t *testing.T) {
	deps := newMainTestDeps(t)
	deps.fileRegular = func(string) bool {
		t.Fatal("file check ran before acknowledgement")
		return true
	}
	deps.configureLogFile = func(string) (func() error, string, error) {
		t.Fatal("log setup ran before acknowledgement")
		return nil, "", nil
	}
	deps.startRuntime = func(context.Context, RuntimeOptions) (runtimeHandle, error) {
		t.Fatal("runtime started before acknowledgement")
		return nil, nil
	}

	exitCode := mainWithDeps(context.Background(), []string{"WORKFLOW.md"}, deps)

	if exitCode != 1 {
		t.Fatalf("exit = %d, want 1", exitCode)
	}
	if got := bufferString(t, deps.stderr); !strings.Contains(got, "This Symphony implementation is a low key engineering preview.") ||
		!strings.Contains(got, "Codex will run without any guardrails.") ||
		!strings.Contains(got, testAckFlag) {
		t.Fatalf("stderr missing acknowledgement banner:\n%s", got)
	}
	if got := bufferString(t, deps.stdout); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
}

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		workflow string
		logsRoot string
		port     *int
		wantErr  bool
	}{
		{
			name:     "default workflow",
			args:     []string{testAckFlag},
			workflow: "WORKFLOW.md",
		},
		{
			name:     "explicit workflow",
			args:     []string{testAckFlag, "custom/WORKFLOW.md"},
			workflow: "custom/WORKFLOW.md",
		},
		{
			name:     "flags after workflow",
			args:     []string{testAckFlag, "custom/WORKFLOW.md", "--port", "0", "--logs-root", "logs"},
			workflow: "custom/WORKFLOW.md",
			logsRoot: "logs",
			port:     ptrInt(0),
		},
		{
			name:     "repeated values last wins",
			args:     []string{testAckFlag, "--port", "1234", "--port", "0", "--logs-root", "old", "--logs-root", "new", "WORKFLOW.md"},
			workflow: "WORKFLOW.md",
			logsRoot: "new",
			port:     ptrInt(0),
		},
		{name: "unknown flag", args: []string{testAckFlag, "--bogus"}, wantErr: true},
		{name: "extra positional", args: []string{testAckFlag, "one", "two"}, wantErr: true},
		{name: "blank logs root", args: []string{testAckFlag, "--logs-root", "   "}, wantErr: true},
		{name: "missing logs root", args: []string{testAckFlag, "--logs-root"}, wantErr: true},
		{name: "logs root value is another flag", args: []string{testAckFlag, "--logs-root", "--port", "0"}, wantErr: true},
		{name: "invalid port", args: []string{testAckFlag, "--port", "abc"}, wantErr: true},
		{name: "negative port", args: []string{testAckFlag, "--port", "-1"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCLIArgs(tt.args)
			if tt.wantErr {
				if !errors.Is(err, errUsage) {
					t.Fatalf("parse err = %v, want errUsage", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse returned error: %v", err)
			}
			if got.workflowPath != tt.workflow {
				t.Fatalf("workflow = %q, want %q", got.workflowPath, tt.workflow)
			}
			if got.logsRoot != tt.logsRoot {
				t.Fatalf("logsRoot = %q, want %q", got.logsRoot, tt.logsRoot)
			}
			if !equalOptionalInt(got.portOverride, tt.port) {
				t.Fatalf("port = %v, want %v", got.portOverride, tt.port)
			}
		})
	}
}

func TestMainHandlesWorkflowStartupAndShutdown(t *testing.T) {
	t.Run("missing workflow file", func(t *testing.T) {
		deps := newMainTestDeps(t)
		deps.fileRegular = func(string) bool { return false }

		exitCode := mainWithDeps(context.Background(), []string{testAckFlag, "missing/WORKFLOW.md"}, deps)

		if exitCode != 1 {
			t.Fatalf("exit = %d, want 1", exitCode)
		}
		expanded := absPath(t, "missing/WORKFLOW.md")
		if got := bufferString(t, deps.stderr); !strings.Contains(got, "Workflow file not found: "+expanded) {
			t.Fatalf("stderr = %q, want missing workflow path %q", got, expanded)
		}
		if got := bufferString(t, deps.stdout); strings.Contains(got, "app_status=offline") {
			t.Fatalf("startup failure rendered offline frame:\n%s", got)
		}
	})

	t.Run("startup error includes workflow path and no offline frame", func(t *testing.T) {
		deps := newMainTestDeps(t)
		deps.startRuntime = func(context.Context, RuntimeOptions) (runtimeHandle, error) {
			return nil, errors.New("boom")
		}

		exitCode := mainWithDeps(context.Background(), []string{testAckFlag, "WORKFLOW.md"}, deps)

		if exitCode != 1 {
			t.Fatalf("exit = %d, want 1", exitCode)
		}
		expanded := absPath(t, "WORKFLOW.md")
		if got := bufferString(t, deps.stderr); !strings.Contains(got, "Failed to start Symphony with workflow "+expanded+": boom") {
			t.Fatalf("stderr = %q, want startup error", got)
		}
		if got := bufferString(t, deps.stdout); strings.Contains(got, "app_status=offline") {
			t.Fatalf("startup failure rendered offline frame:\n%s", got)
		}
	})

	t.Run("normal shutdown closes runtime and renders minimal offline frame", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		deps := newMainTestDeps(t)
		rt := &fakeRuntime{settings: config.Settings{Observability: config.ObservabilitySettings{DashboardEnabled: false}}}
		var gotOptions RuntimeOptions
		var gotLogRoot string
		deps.configureLogFile = func(root string) (func() error, string, error) {
			gotLogRoot = root
			return func() error { return nil }, filepath.Join(root, "log", "symphony.log"), nil
		}
		deps.startRuntime = func(_ context.Context, opts RuntimeOptions) (runtimeHandle, error) {
			gotOptions = opts
			cancel()
			return rt, nil
		}

		exitCode := mainWithDeps(ctx, []string{testAckFlag, "WORKFLOW.md", "--logs-root", "logs", "--port", "0"}, deps)

		if exitCode != 0 {
			t.Fatalf("exit = %d, want 0; stderr=%s", exitCode, bufferString(t, deps.stderr))
		}
		if gotOptions.WorkflowPath != absPath(t, "WORKFLOW.md") {
			t.Fatalf("workflow option = %q, want expanded path", gotOptions.WorkflowPath)
		}
		if gotOptions.ServerPortOverride == nil || *gotOptions.ServerPortOverride != 0 {
			t.Fatalf("server port override = %v, want 0", gotOptions.ServerPortOverride)
		}
		if gotLogRoot != absPath(t, "logs") {
			t.Fatalf("logs root = %q, want expanded", gotLogRoot)
		}
		if !rt.closed {
			t.Fatal("runtime was not closed")
		}
		stdout := bufferString(t, deps.stdout)
		wantFrame := dashboard.RenderOffline() + "\n"
		if stdout != wantFrame {
			t.Fatalf("offline stdout mismatch\n--- got ---\n%s\n--- want ---\n%s", stdout, wantFrame)
		}
		if strings.Count(stdout, "app_status=offline") != 1 {
			t.Fatalf("offline stdout rendered %d offline markers, want 1:\n%s", strings.Count(stdout, "app_status=offline"), stdout)
		}
		for _, forbidden := range []string{"Timestamp:", "Running", "Backoff queue"} {
			if strings.Contains(stdout, forbidden) {
				t.Fatalf("offline stdout contains %q:\n%s", forbidden, stdout)
			}
		}
	})

	t.Run("close error returns nonzero", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		deps := newMainTestDeps(t)
		deps.startRuntime = func(context.Context, RuntimeOptions) (runtimeHandle, error) {
			cancel()
			return &fakeRuntime{closeErr: errors.New("close boom")}, nil
		}

		exitCode := mainWithDeps(ctx, []string{testAckFlag, "WORKFLOW.md"}, deps)

		if exitCode != 1 {
			t.Fatalf("exit = %d, want 1", exitCode)
		}
		if got := bufferString(t, deps.stderr); !strings.Contains(got, "close boom") {
			t.Fatalf("stderr = %q, want close error", got)
		}
	})
}

func TestConfigureLogFileUsesExpandedRootAndRestoresLogger(t *testing.T) {
	var restored bytes.Buffer
	log.SetOutput(&restored)
	restore, path, err := configureLogFile(filepath.Join(t.TempDir(), "custom"))
	if err != nil {
		t.Fatalf("configureLogFile returned error: %v", err)
	}
	log.Print("hello file")
	if err := restore(); err != nil {
		t.Fatalf("restore returned error: %v", err)
	}
	log.Print("hello buffer")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "hello file") {
		t.Fatalf("log file = %q, want file log", content)
	}
	if !strings.Contains(restored.String(), "hello buffer") {
		t.Fatalf("restored logger output = %q, want buffer log", restored.String())
	}
	if filepath.Base(path) != "symphony.log" || filepath.Base(filepath.Dir(path)) != "log" {
		t.Fatalf("log path = %s, want <root>/log/symphony.log", path)
	}
}

func bufferString(t *testing.T, writer any) string {
	t.Helper()
	buffer, ok := writer.(*bytes.Buffer)
	if !ok {
		t.Fatalf("writer = %T, want *bytes.Buffer", writer)
	}
	return buffer.String()
}

func newMainTestDeps(t *testing.T) mainDeps {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return mainDeps{
		stdout: stdout,
		stderr: stderr,
		fileRegular: func(string) bool {
			return true
		},
		configureLogFile: func(string) (func() error, string, error) {
			return func() error { return nil }, "", nil
		},
		startRuntime: func(context.Context, RuntimeOptions) (runtimeHandle, error) {
			return &fakeRuntime{}, nil
		},
	}
}

type fakeRuntime struct {
	settings     config.Settings
	snapshot     domain.Snapshot
	dashboardURL string
	closeErr     error
	closed       bool
}

func (r *fakeRuntime) Close() error {
	r.closed = true
	return r.closeErr
}

func (r *fakeRuntime) Snapshot() domain.Snapshot { return r.snapshot }
func (r *fakeRuntime) DashboardURL() string      { return r.dashboardURL }
func (r *fakeRuntime) Settings() config.Settings { return r.settings }

func ptrInt(value int) *int {
	return &value
}

func equalOptionalInt(a, b *int) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func absPath(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}
	return abs
}

func TestDefaultLogFile(t *testing.T) {
	root := filepath.Join("tmp", "symphony-logs")
	want := filepath.Join(root, "log", "symphony.log")
	if got := DefaultLogFile(root); got != want {
		t.Fatalf("DefaultLogFile = %q, want %q", got, want)
	}
	if got := fmt.Sprint(DefaultLogFile("")); got == "" {
		t.Fatal("DefaultLogFile with empty root returned empty path")
	}
}
