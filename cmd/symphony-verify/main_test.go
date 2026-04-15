package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/cli"
	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
	"github.com/Miss-you/go-symphony/internal/trackers/memory"
)

func TestLinearProbeUsesFakeReaderAndRendersReport(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "linear", "api_key: token\n  project_slug: PROJ")
	deps := newVerifyTestDeps(t)
	deps.newLinearReader = func(settings config.ProviderSettings) (tracker.TrackerReader, error) {
		if settings.Project != "PROJ" {
			t.Fatalf("project = %q, want PROJ", settings.Project)
		}
		return memory.NewReader([]domain.WorkItem{
			{ID: "item-1", Identifier: "ABC-1", Title: "One", State: "Todo"},
			{ID: "item-2", Identifier: "ABC-2", Title: "Two", State: "Done"},
		}), nil
	}
	deps.startRuntime = func(context.Context, cli.RuntimeOptions) (runtimeHandle, error) {
		t.Fatal("linear probe started runtime")
		return nil, nil
	}

	exit := mainWithDeps(context.Background(), []string{"linear", "--limit", "1", "--refresh-id", "item-1", workflowPath}, deps)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr=%s", exit, stderrString(t, deps))
	}
	stdout := stdoutString(t, deps)
	for _, want := range []string{
		"Linear probe",
		"project: PROJ",
		"candidates: 2",
		"ABC-1",
		"terminal: 1",
		"refresh: 1",
		"... 1 more",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestLinearProbeCanFilterSingleIssue(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "linear", "api_key: token\n  project_slug: PROJ")
	deps := newVerifyTestDeps(t)
	deps.newLinearReader = func(config.ProviderSettings) (tracker.TrackerReader, error) {
		return memory.NewReader([]domain.WorkItem{
			{ID: "item-1", Identifier: "ABC-1", Title: "One", State: "Todo"},
			{ID: "item-2", Identifier: "ABC-2", Title: "Two", State: "Todo"},
			{ID: "item-3", Identifier: "ABC-3", Title: "Done", State: "Done"},
		}), nil
	}

	exit := mainWithDeps(context.Background(), []string{"linear", "--only-issue", "ABC-2", "--refresh-id", "item-2", workflowPath}, deps)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr=%s", exit, stderrString(t, deps))
	}
	stdout := stdoutString(t, deps)
	for _, want := range []string{
		"candidates: 1",
		"ABC-2",
		"terminal: 0",
		"refresh: 1",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, notWant := range []string{"ABC-1", "ABC-3"} {
		if strings.Contains(stdout, notWant) {
			t.Fatalf("stdout unexpectedly contains %q:\n%s", notWant, stdout)
		}
	}
}

func TestLinearProbeRejectsNonLinearBeforeCreatingReader(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "memory", "")
	deps := newVerifyTestDeps(t)
	deps.newLinearReader = func(config.ProviderSettings) (tracker.TrackerReader, error) {
		t.Fatal("created Linear reader for non-Linear workflow")
		return nil, nil
	}

	exit := mainWithDeps(context.Background(), []string{"linear", workflowPath}, deps)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	if got := stderrString(t, deps); !strings.Contains(got, "linear probe requires tracker.kind=linear") {
		t.Fatalf("stderr = %q, want provider error", got)
	}
}

func TestRunRequiresAcknowledgementAndOnlyIssueBeforeStartup(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "linear", "api_key: token\n  project_slug: PROJ")
	deps := newVerifyTestDeps(t)
	deps.startRuntime = func(context.Context, cli.RuntimeOptions) (runtimeHandle, error) {
		t.Fatal("runtime started before required run flags")
		return nil, nil
	}

	exit := mainWithDeps(context.Background(), []string{"run", "--only-issue", "ABC-1", workflowPath}, deps)
	if exit != 1 {
		t.Fatalf("missing ack exit = %d, want 1", exit)
	}
	if !strings.Contains(stderrString(t, deps), ackFlag) {
		t.Fatalf("stderr missing ack flag: %s", stderrString(t, deps))
	}

	stderrBuffer(t, deps).Reset()
	exit = mainWithDeps(context.Background(), []string{"run", ackFlag, workflowPath}, deps)
	if exit != 1 {
		t.Fatalf("missing only issue exit = %d, want 1", exit)
	}
	if !strings.Contains(stderrString(t, deps), "--only-issue") {
		t.Fatalf("stderr missing only-issue error: %s", stderrString(t, deps))
	}
}

func TestRunStartsRuntimeWithFilteredReader(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "linear", "api_key: token\n  project_slug: PROJ")
	deps := newVerifyTestDeps(t)
	deps.newLinearReader = func(config.ProviderSettings) (tracker.TrackerReader, error) {
		return memory.NewReader([]domain.WorkItem{
			{ID: "item-1", Identifier: "ABC-1", State: "Todo"},
			{ID: "item-2", Identifier: "ABC-2", State: "Todo"},
		}), nil
	}
	var gotOptions cli.RuntimeOptions
	deps.startRuntime = func(_ context.Context, opts cli.RuntimeOptions) (runtimeHandle, error) {
		gotOptions = opts
		return fakeRuntime{dashboardURL: "http://127.0.0.1:1234/"}, nil
	}

	exit := mainWithDeps(context.Background(), []string{"run", ackFlag, "--only-issue", "ABC-2", "--port", "0", "--timeout", "1ns", workflowPath}, deps)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr=%s", exit, stderrString(t, deps))
	}
	if gotOptions.Reader == nil {
		t.Fatal("runtime reader is nil")
	}
	items, err := gotOptions.Reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("filtered reader ListCandidates: %v", err)
	}
	if len(items) != 1 || items[0].Identifier != "ABC-2" {
		t.Fatalf("filtered candidates = %#v, want only ABC-2", items)
	}
	if gotOptions.ServerPortOverride == nil || *gotOptions.ServerPortOverride != 0 {
		t.Fatalf("port override = %v, want 0", gotOptions.ServerPortOverride)
	}
	if !strings.Contains(stdoutString(t, deps), "http://127.0.0.1:1234/") {
		t.Fatalf("stdout missing dashboard URL: %s", stdoutString(t, deps))
	}
}

func TestLinearProbePackageDoesNotDependOnRuntimePackages(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "list", "-deps", "github.com/Miss-you/go-symphony/internal/verify/linearprobe")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list linearprobe deps: %v\n%s", err, output)
	}
	deps := string(output)
	for _, forbidden := range []string{
		"internal/cli",
		"internal/codex",
		"internal/orchestrator",
		"internal/workspace",
	} {
		if strings.Contains(deps, forbidden) {
			t.Fatalf("linearprobe package depends on forbidden runtime package %q\n%s", forbidden, deps)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{},
		{"bogus"},
		{"linear", "--limit"},
		{"linear", "--limit", "-1"},
		{"linear", "one", "two"},
		{"run", ackFlag, "--only-issue"},
		{"run", ackFlag, "--only-issue", "   "},
		{"run", ackFlag, "--only-issue", "ABC-1", "--timeout", "bad"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Parallel()
			deps := newVerifyTestDeps(t)
			exit := mainWithDeps(context.Background(), args, deps)
			if exit != 1 {
				t.Fatalf("exit = %d, want 1; stdout=%s stderr=%s", exit, stdoutString(t, deps), stderrString(t, deps))
			}
		})
	}
}

func newVerifyTestDeps(t *testing.T) verifyDeps {
	t.Helper()
	return verifyDeps{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		openStore: func(path string) (*config.Store, error) {
			return config.NewStore(config.WithWorkflowPath(path))
		},
		newLinearReader: func(config.ProviderSettings) (tracker.TrackerReader, error) {
			return nil, errors.New("unexpected real reader")
		},
		startRuntime: func(context.Context, cli.RuntimeOptions) (runtimeHandle, error) {
			return fakeRuntime{}, nil
		},
	}
}

func stdoutString(t *testing.T, deps verifyDeps) string {
	t.Helper()
	return stdoutBuffer(t, deps).String()
}

func stderrString(t *testing.T, deps verifyDeps) string {
	t.Helper()
	return stderrBuffer(t, deps).String()
}

func stdoutBuffer(t *testing.T, deps verifyDeps) *bytes.Buffer {
	t.Helper()
	buffer, ok := deps.stdout.(*bytes.Buffer)
	if !ok {
		t.Fatalf("stdout = %T, want *bytes.Buffer", deps.stdout)
	}
	return buffer
}

func stderrBuffer(t *testing.T, deps verifyDeps) *bytes.Buffer {
	t.Helper()
	buffer, ok := deps.stderr.(*bytes.Buffer)
	if !ok {
		t.Fatalf("stderr = %T, want *bytes.Buffer", deps.stderr)
	}
	return buffer
}

func writeVerifyWorkflow(t *testing.T, provider, providerExtra string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "WORKFLOW.md")
	content := `---
tracker:
  kind: ` + provider + `
  ` + providerExtra + `
  active_states: ["Todo"]
  terminal_states: ["Done"]
agent:
  max_concurrent_agents: 1
  max_turns: 1
  max_retry_backoff_ms: 1000
codex:
  command: codex app-server
  approval_policy: never
  turn_timeout_ms: 1000
  read_timeout_ms: 1000
  stall_timeout_ms: 0
hooks:
  timeout_ms: 1000
---
Prompt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	return path
}

type fakeRuntime struct {
	dashboardURL string
}

func (r fakeRuntime) Close() error { return nil }

func (r fakeRuntime) DashboardURL() string { return r.dashboardURL }

func (r fakeRuntime) Snapshot() domain.Snapshot {
	return domain.Snapshot{
		Running: []domain.ActiveRun{{ItemID: "item-2", ItemIdentifier: "ABC-2"}},
		CodexTotals: domain.CodexTotals{
			TotalTokens: 3,
		},
	}
}

var _ runtimeHandle = fakeRuntime{}

func TestRunPropagatesRuntimeStartupError(t *testing.T) {
	t.Parallel()

	workflowPath := writeVerifyWorkflow(t, "linear", "api_key: token\n  project_slug: PROJ")
	deps := newVerifyTestDeps(t)
	deps.newLinearReader = func(config.ProviderSettings) (tracker.TrackerReader, error) {
		return memory.NewReader(nil), nil
	}
	deps.startRuntime = func(context.Context, cli.RuntimeOptions) (runtimeHandle, error) {
		return nil, errors.New("boom")
	}

	exit := mainWithDeps(context.Background(), []string{"run", ackFlag, "--only-issue", "ABC-1", workflowPath}, deps)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	if !strings.Contains(stderrString(t, deps), "failed to start verification runtime: boom") {
		t.Fatalf("stderr = %q, want startup error", stderrString(t, deps))
	}
}

func TestRunParsesZeroTimeout(t *testing.T) {
	t.Parallel()

	args, err := parseRunArgs([]string{ackFlag, "--only-issue", "ABC-1", "--timeout", "0"})
	if err != nil {
		t.Fatalf("parseRunArgs: %v", err)
	}
	if args.timeout != 0 {
		t.Fatalf("timeout = %s, want 0", args.timeout)
	}
	if args.workflowPath != "WORKFLOW.md" {
		t.Fatalf("workflow = %q, want WORKFLOW.md", args.workflowPath)
	}
}

func TestParseDurationTimeout(t *testing.T) {
	t.Parallel()

	args, err := parseRunArgs([]string{ackFlag, "--only-issue", "ABC-1", "--timeout=2s"})
	if err != nil {
		t.Fatalf("parseRunArgs: %v", err)
	}
	if args.timeout != 2*time.Second {
		t.Fatalf("timeout = %s, want 2s", args.timeout)
	}
}
