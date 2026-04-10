package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLoadSettingsAppliesDefaultsAndProviderMapping(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "SETTINGS_WORKFLOW.md", `---
tracker:
  kind: linear
  api_key: token
  project_slug: project
---
Prompt body
`)

	got, err := LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	if got.Provider.Kind != ProviderLinear {
		t.Fatalf("provider kind mismatch: got %q want %q", got.Provider.Kind, ProviderLinear)
	}
	if got.Provider.Project != "project" {
		t.Fatalf("provider project mismatch: got %q want %q", got.Provider.Project, "project")
	}
	if got.Provider.APIKey != "token" {
		t.Fatalf("provider api key mismatch: got %q want %q", got.Provider.APIKey, "token")
	}
	if got.Provider.Endpoint != "https://api.linear.app/graphql" {
		t.Fatalf("provider endpoint mismatch: got %q", got.Provider.Endpoint)
	}
	if !slices.Equal(got.Provider.ActiveStates, []string{"Todo", "In Progress"}) {
		t.Fatalf("active states mismatch: got %#v", got.Provider.ActiveStates)
	}
	if !slices.Equal(got.Provider.TerminalStates, []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"}) {
		t.Fatalf("terminal states mismatch: got %#v", got.Provider.TerminalStates)
	}
	if got.Polling.IntervalMS != 30_000 {
		t.Fatalf("polling interval mismatch: got %d want %d", got.Polling.IntervalMS, 30_000)
	}
	if got.Workspace.Root != filepath.Join(os.TempDir(), "symphony_workspaces") {
		t.Fatalf("workspace root mismatch: got %q want default temp root", got.Workspace.Root)
	}
	if got.Agent.MaxConcurrentAgents != 10 {
		t.Fatalf("max concurrent agents mismatch: got %d want %d", got.Agent.MaxConcurrentAgents, 10)
	}
	if got.Agent.MaxTurns != 20 {
		t.Fatalf("max turns mismatch: got %d want %d", got.Agent.MaxTurns, 20)
	}
	if got.Codex.Command != "codex app-server" {
		t.Fatalf("codex command mismatch: got %q", got.Codex.Command)
	}
	if got.Server.Host != "127.0.0.1" {
		t.Fatalf("server host mismatch: got %q", got.Server.Host)
	}
}

func TestParseSettingsResolvesEnvFallbacksAndWorkspaceRoot(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "env-token")
	t.Setenv("LINEAR_ASSIGNEE", "dev@example.com")
	wantRoot := filepath.Join(t.TempDir(), "workspace-from-env")
	t.Setenv("WORKSPACE_ROOT_TOKEN", wantRoot)

	workflow := Workflow{
		Config: map[string]any{
			"tracker": map[string]any{
				"kind":         "linear",
				"project_slug": "project",
			},
			"workspace": map[string]any{
				"root": "$WORKSPACE_ROOT_TOKEN",
			},
		},
	}

	got, err := ParseSettings(workflow)
	if err != nil {
		t.Fatalf("ParseSettings: %v", err)
	}

	if got.Provider.APIKey != "env-token" {
		t.Fatalf("api key mismatch: got %q want %q", got.Provider.APIKey, "env-token")
	}
	if got.Provider.Assignee != "dev@example.com" {
		t.Fatalf("assignee mismatch: got %q want %q", got.Provider.Assignee, "dev@example.com")
	}
	if got.Workspace.Root != wantRoot {
		t.Fatalf("workspace root mismatch: got %q want %q", got.Workspace.Root, wantRoot)
	}
}

func TestParseSettingsAcceptsMemoryProviderWithoutLinearCredentials(t *testing.T) {
	t.Parallel()

	got, err := ParseSettings(Workflow{
		Config: map[string]any{
			"tracker": map[string]any{
				"kind": "memory",
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseSettings: %v", err)
	}

	if got.Provider.Kind != ProviderMemory {
		t.Fatalf("provider kind mismatch: got %q want %q", got.Provider.Kind, ProviderMemory)
	}
}

func TestParseSettingsExpandsAndFallsBackWorkspaceRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EMPTY_WORKSPACE_ROOT", "")

	defaultRoot := filepath.Join(os.TempDir(), "symphony_workspaces")

	testCases := []struct {
		name string
		root string
		want string
	}{
		{
			name: "home expansion",
			root: "~/repo-workspace",
			want: filepath.Join(home, "repo-workspace"),
		},
		{
			name: "missing env falls back to default",
			root: "$MISSING_WORKSPACE_ROOT",
			want: defaultRoot,
		},
		{
			name: "empty env falls back to default",
			root: "$EMPTY_WORKSPACE_ROOT",
			want: defaultRoot,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workflow := Workflow{
				Config: map[string]any{
					"tracker": map[string]any{
						"kind":         "linear",
						"api_key":      "token",
						"project_slug": "project",
					},
					"workspace": map[string]any{
						"root": tc.root,
					},
				},
			}

			got, err := ParseSettings(workflow)
			if err != nil {
				t.Fatalf("ParseSettings: %v", err)
			}
			if got.Workspace.Root != tc.want {
				t.Fatalf("workspace root mismatch: got %q want %q", got.Workspace.Root, tc.want)
			}
		})
	}
}

func TestParseSettingsRejectsInvalidProviderConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		config    map[string]any
		wantText string
	}{
		{
			name: "missing provider kind",
			config: map[string]any{
				"tracker": map[string]any{},
			},
			wantText: "provider.kind",
		},
		{
			name: "unsupported provider kind",
			config: map[string]any{
				"tracker": map[string]any{
					"kind": "github",
				},
			},
			wantText: "provider.kind",
		},
		{
			name: "missing linear project",
			config: map[string]any{
				"tracker": map[string]any{
					"kind":    "linear",
					"api_key": "token",
				},
			},
			wantText: "provider.project",
		},
		{
			name: "blank codex command",
			config: map[string]any{
				"tracker": map[string]any{
					"kind":         "linear",
					"api_key":      "token",
					"project_slug": "project",
				},
				"codex": map[string]any{
					"command": "",
				},
			},
			wantText: "codex.command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseSettings(Workflow{Config: tc.config})
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); got == "" || !containsString(got, tc.wantText) {
				t.Fatalf("error mismatch: got %q want substring %q", got, tc.wantText)
			}
		})
	}
}

func TestNewStoreFailsWhenTypedSettingsInvalid(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "INVALID_TYPED_SETTINGS.md", `---
tracker:
  kind: linear
---
Prompt body
`)

	_, err := NewStore(WithWorkflowPath(path), withTickChannel(make(chan time.Time)), withLogf(func(string, ...any) {}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStoreCurrentSettingsKeepsAtomicLastKnownGoodSnapshotOnInvalidTypedReload(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "VALID_TYPED_SETTINGS.md", `---
tracker:
  kind: linear
  api_key: token
  project_slug: project
---
Initial prompt
`)
	ticks := make(chan time.Time, 1)

	store, err := NewStore(
		WithWorkflowPath(path),
		withTickChannel(ticks),
		withLogf(func(string, ...any) {}),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := overwriteFile(path, `---
tracker:
  kind: linear
  api_key: token
polling:
  interval_ms: 0
---
Broken prompt
`); err != nil {
		t.Fatalf("overwrite workflow: %v", err)
	}

	if err := store.ForceReload(); err == nil {
		t.Fatal("expected reload error")
	}

	workflow, err := store.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if workflow.PromptTemplate != "Initial prompt" {
		t.Fatalf("workflow snapshot mismatch: got %q want %q", workflow.PromptTemplate, "Initial prompt")
	}

	settings, err := store.CurrentSettings()
	if err != nil {
		t.Fatalf("CurrentSettings: %v", err)
	}
	if settings.Provider.Project != "project" {
		t.Fatalf("settings snapshot mismatch: got %q want %q", settings.Provider.Project, "project")
	}
}

func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
