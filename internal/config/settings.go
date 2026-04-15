package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderKind string

const (
	ProviderLinear ProviderKind = "linear"
	ProviderMemory ProviderKind = "memory"
)

type Settings struct {
	Provider      ProviderSettings
	Polling       PollingSettings
	Workspace     WorkspaceSettings
	Worker        WorkerSettings
	Agent         AgentSettings
	Codex         CodexSettings
	Hooks         HookSettings
	Observability ObservabilitySettings
	Server        ServerSettings
}

type ProviderSettings struct {
	Kind           ProviderKind
	Endpoint       string
	APIKey         string
	Project        string
	Assignee       string
	ActiveStates   []string
	TerminalStates []string
}

type PollingSettings struct {
	IntervalMS int
}

type WorkspaceSettings struct {
	Root string
}

type WorkerSettings struct {
	SSHHosts                   []string
	MaxConcurrentAgentsPerHost *int
}

type AgentSettings struct {
	MaxConcurrentAgents        int
	MaxTurns                   int
	MaxRetryBackoffMS          int
	MaxConcurrentAgentsByState map[string]int
}

type CodexSettings struct {
	Command           string
	ApprovalPolicy    any
	ThreadSandbox     string
	TurnSandboxPolicy map[string]any
	TurnTimeoutMS     int
	ReadTimeoutMS     int
	StallTimeoutMS    int
}

type HookSettings struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	TimeoutMS    int
}

type ObservabilitySettings struct {
	DashboardEnabled bool
	RefreshMS        int
	RenderIntervalMS int
}

type ServerSettings struct {
	Port *int
	Host string
}

type legacyWorkflowConfig struct {
	Tracker       legacyTrackerConfig       `yaml:"tracker"`
	Polling       legacyPollingConfig       `yaml:"polling"`
	Workspace     legacyWorkspaceConfig     `yaml:"workspace"`
	Worker        legacyWorkerConfig        `yaml:"worker"`
	Agent         legacyAgentConfig         `yaml:"agent"`
	Codex         legacyCodexConfig         `yaml:"codex"`
	Hooks         legacyHookConfig          `yaml:"hooks"`
	Observability legacyObservabilityConfig `yaml:"observability"`
	Server        legacyServerConfig        `yaml:"server"`
}

type legacyTrackerConfig struct {
	Kind           string   `yaml:"kind"`
	Endpoint       string   `yaml:"endpoint"`
	APIKey         string   `yaml:"api_key"`
	ProjectSlug    string   `yaml:"project_slug"`
	Assignee       string   `yaml:"assignee"`
	ActiveStates   []string `yaml:"active_states"`
	TerminalStates []string `yaml:"terminal_states"`
}

type legacyPollingConfig struct {
	IntervalMS int `yaml:"interval_ms"`
}

type legacyWorkspaceConfig struct {
	Root string `yaml:"root"`
}

type legacyWorkerConfig struct {
	SSHHosts                   []string `yaml:"ssh_hosts"`
	MaxConcurrentAgentsPerHost *int     `yaml:"max_concurrent_agents_per_host"`
}

type legacyAgentConfig struct {
	MaxConcurrentAgents        int            `yaml:"max_concurrent_agents"`
	MaxTurns                   int            `yaml:"max_turns"`
	MaxRetryBackoffMS          int            `yaml:"max_retry_backoff_ms"`
	MaxConcurrentAgentsByState map[string]int `yaml:"max_concurrent_agents_by_state"`
}

type legacyCodexConfig struct {
	Command           string         `yaml:"command"`
	ApprovalPolicy    any            `yaml:"approval_policy"`
	ThreadSandbox     string         `yaml:"thread_sandbox"`
	TurnSandboxPolicy map[string]any `yaml:"turn_sandbox_policy"`
	TurnTimeoutMS     int            `yaml:"turn_timeout_ms"`
	ReadTimeoutMS     int            `yaml:"read_timeout_ms"`
	StallTimeoutMS    int            `yaml:"stall_timeout_ms"`
}

type legacyHookConfig struct {
	AfterCreate  string `yaml:"after_create"`
	BeforeRun    string `yaml:"before_run"`
	AfterRun     string `yaml:"after_run"`
	BeforeRemove string `yaml:"before_remove"`
	TimeoutMS    int    `yaml:"timeout_ms"`
}

type legacyObservabilityConfig struct {
	DashboardEnabled bool `yaml:"dashboard_enabled"`
	RefreshMS        int  `yaml:"refresh_ms"`
	RenderIntervalMS int  `yaml:"render_interval_ms"`
}

type legacyServerConfig struct {
	Port *int   `yaml:"port"`
	Host string `yaml:"host"`
}

func LoadSettings(path string) (Settings, error) {
	workflow, err := Load(path)
	if err != nil {
		return Settings{}, err
	}
	return ParseSettings(workflow)
}

func ParseSettings(workflow Workflow) (Settings, error) {
	legacy, err := decodeLegacyWorkflowConfig(workflow.Config)
	if err != nil {
		return Settings{}, err
	}

	settings := Settings{
		Provider: ProviderSettings{
			Kind:           ProviderKind(strings.TrimSpace(legacy.Tracker.Kind)),
			Endpoint:       strings.TrimSpace(legacy.Tracker.Endpoint),
			APIKey:         resolveSecretSetting(legacy.Tracker.APIKey, "LINEAR_API_KEY"),
			Project:        resolveEnvReferenceSetting(legacy.Tracker.ProjectSlug),
			Assignee:       resolveSecretSetting(legacy.Tracker.Assignee, "LINEAR_ASSIGNEE"),
			ActiveStates:   append([]string(nil), legacy.Tracker.ActiveStates...),
			TerminalStates: append([]string(nil), legacy.Tracker.TerminalStates...),
		},
		Polling: PollingSettings{
			IntervalMS: legacy.Polling.IntervalMS,
		},
		Workspace: WorkspaceSettings{
			Root: resolveWorkspaceRoot(legacy.Workspace.Root),
		},
		Worker: WorkerSettings{
			SSHHosts:                   append([]string(nil), legacy.Worker.SSHHosts...),
			MaxConcurrentAgentsPerHost: cloneOptionalInt(legacy.Worker.MaxConcurrentAgentsPerHost),
		},
		Agent: AgentSettings{
			MaxConcurrentAgents:        legacy.Agent.MaxConcurrentAgents,
			MaxTurns:                   legacy.Agent.MaxTurns,
			MaxRetryBackoffMS:          legacy.Agent.MaxRetryBackoffMS,
			MaxConcurrentAgentsByState: normalizeStateLimitKeys(legacy.Agent.MaxConcurrentAgentsByState),
		},
		Codex: CodexSettings{
			Command:           legacy.Codex.Command,
			ApprovalPolicy:    normalizeConfigValue(legacy.Codex.ApprovalPolicy),
			ThreadSandbox:     strings.TrimSpace(legacy.Codex.ThreadSandbox),
			TurnSandboxPolicy: normalizeOptionalMap(legacy.Codex.TurnSandboxPolicy),
			TurnTimeoutMS:     legacy.Codex.TurnTimeoutMS,
			ReadTimeoutMS:     legacy.Codex.ReadTimeoutMS,
			StallTimeoutMS:    legacy.Codex.StallTimeoutMS,
		},
		Hooks: HookSettings{
			AfterCreate:  legacy.Hooks.AfterCreate,
			BeforeRun:    legacy.Hooks.BeforeRun,
			AfterRun:     legacy.Hooks.AfterRun,
			BeforeRemove: legacy.Hooks.BeforeRemove,
			TimeoutMS:    legacy.Hooks.TimeoutMS,
		},
		Observability: ObservabilitySettings{
			DashboardEnabled: legacy.Observability.DashboardEnabled,
			RefreshMS:        legacy.Observability.RefreshMS,
			RenderIntervalMS: legacy.Observability.RenderIntervalMS,
		},
		Server: ServerSettings{
			Port: cloneOptionalInt(legacy.Server.Port),
			Host: strings.TrimSpace(legacy.Server.Host),
		},
	}

	if err := validateSettings(settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func decodeLegacyWorkflowConfig(raw map[string]any) (legacyWorkflowConfig, error) {
	cfg := defaultLegacyWorkflowConfig()
	if len(raw) == 0 {
		return cfg, nil
	}

	normalized, ok := dropNilConfigValue(normalizeConfigValue(raw)).(map[string]any)
	if !ok {
		return legacyWorkflowConfig{}, fmt.Errorf("invalid workflow config: root must be a map")
	}

	content, err := yaml.Marshal(normalized)
	if err != nil {
		return legacyWorkflowConfig{}, fmt.Errorf("invalid workflow config: %w", err)
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return legacyWorkflowConfig{}, fmt.Errorf("invalid workflow config: %w", err)
	}

	return cfg, nil
}

func defaultLegacyWorkflowConfig() legacyWorkflowConfig {
	return legacyWorkflowConfig{
		Tracker: legacyTrackerConfig{
			Endpoint:       "https://api.linear.app/graphql",
			ActiveStates:   []string{"Todo", "In Progress"},
			TerminalStates: []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"},
		},
		Polling: legacyPollingConfig{
			IntervalMS: 30_000,
		},
		Workspace: legacyWorkspaceConfig{
			Root: filepath.Join(os.TempDir(), "symphony_workspaces"),
		},
		Worker: legacyWorkerConfig{
			SSHHosts: []string{},
		},
		Agent: legacyAgentConfig{
			MaxConcurrentAgents:        10,
			MaxTurns:                   20,
			MaxRetryBackoffMS:          300_000,
			MaxConcurrentAgentsByState: map[string]int{},
		},
		Codex: legacyCodexConfig{
			Command:        "codex app-server",
			ApprovalPolicy: map[string]any{"reject": map[string]any{"sandbox_approval": true, "rules": true, "mcp_elicitations": true}},
			ThreadSandbox:  "workspace-write",
			TurnTimeoutMS:  3_600_000,
			ReadTimeoutMS:  5_000,
			StallTimeoutMS: 300_000,
		},
		Hooks: legacyHookConfig{
			TimeoutMS: 60_000,
		},
		Observability: legacyObservabilityConfig{
			DashboardEnabled: true,
			RefreshMS:        1_000,
			RenderIntervalMS: 16,
		},
		Server: legacyServerConfig{
			Host: "127.0.0.1",
		},
	}
}

func validateSettings(settings Settings) error {
	if settings.Provider.Kind == "" {
		return fmt.Errorf("provider.kind must not be blank")
	}
	if settings.Provider.Kind != ProviderLinear && settings.Provider.Kind != ProviderMemory {
		return fmt.Errorf("provider.kind must be one of [%s %s]", ProviderLinear, ProviderMemory)
	}

	if settings.Provider.Kind == ProviderLinear {
		if strings.TrimSpace(settings.Provider.APIKey) == "" {
			return fmt.Errorf("provider.api_key is required for linear provider")
		}
		if strings.TrimSpace(settings.Provider.Project) == "" {
			return fmt.Errorf("provider.project is required for linear provider")
		}
	}
	if settings.Codex.Command == "" {
		return fmt.Errorf("codex.command must not be blank")
	}

	if err := requirePositive("polling.interval_ms", settings.Polling.IntervalMS); err != nil {
		return err
	}
	if err := requirePositive("agent.max_concurrent_agents", settings.Agent.MaxConcurrentAgents); err != nil {
		return err
	}
	if err := requirePositive("agent.max_turns", settings.Agent.MaxTurns); err != nil {
		return err
	}
	if err := requirePositive("agent.max_retry_backoff_ms", settings.Agent.MaxRetryBackoffMS); err != nil {
		return err
	}
	if settings.Worker.MaxConcurrentAgentsPerHost != nil {
		if err := requirePositive("worker.max_concurrent_agents_per_host", *settings.Worker.MaxConcurrentAgentsPerHost); err != nil {
			return err
		}
	}
	if err := requirePositive("codex.turn_timeout_ms", settings.Codex.TurnTimeoutMS); err != nil {
		return err
	}
	if err := requirePositive("codex.read_timeout_ms", settings.Codex.ReadTimeoutMS); err != nil {
		return err
	}
	if settings.Codex.StallTimeoutMS < 0 {
		return fmt.Errorf("codex.stall_timeout_ms must be >= 0")
	}
	if err := requirePositive("hooks.timeout_ms", settings.Hooks.TimeoutMS); err != nil {
		return err
	}
	if err := requirePositive("observability.refresh_ms", settings.Observability.RefreshMS); err != nil {
		return err
	}
	if err := requirePositive("observability.render_interval_ms", settings.Observability.RenderIntervalMS); err != nil {
		return err
	}
	if settings.Server.Port != nil && *settings.Server.Port < 0 {
		return fmt.Errorf("server.port must be >= 0")
	}

	for state, limit := range settings.Agent.MaxConcurrentAgentsByState {
		if strings.TrimSpace(state) == "" {
			return fmt.Errorf("agent.max_concurrent_agents_by_state keys must not be blank")
		}
		if limit <= 0 {
			return fmt.Errorf("agent.max_concurrent_agents_by_state values must be positive")
		}
	}

	switch settings.Codex.ApprovalPolicy.(type) {
	case string, map[string]any:
	default:
		return fmt.Errorf("codex.approval_policy must be a string or map")
	}
	if settings.Codex.TurnSandboxPolicy == nil {
		return nil
	}
	return nil
}

func requirePositive(field string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be > 0", field)
	}
	return nil
}

func normalizeConfigValue(value any) any {
	return normalizeYAMLValue(value)
}

func dropNilConfigValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			normalizedChild := dropNilConfigValue(child)
			if normalizedChild == nil {
				continue
			}
			normalized[key] = normalizedChild
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, child := range typed {
			normalized = append(normalized, dropNilConfigValue(child))
		}
		return normalized
	default:
		return typed
	}
}

func resolveSecretSetting(value, fallbackEnvVar string) string {
	fallback := normalizeSecretValue(os.Getenv(fallbackEnvVar))
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	if envName, ok := envReferenceName(value); ok {
		envValue, exists := os.LookupEnv(envName)
		if !exists {
			return fallback
		}
		return normalizeSecretValue(envValue)
	}

	return normalizeSecretValue(value)
}

func resolveEnvReferenceSetting(value string) string {
	trimmed := strings.TrimSpace(value)
	if envName, ok := envReferenceName(trimmed); ok {
		envValue, exists := os.LookupEnv(envName)
		if !exists {
			return ""
		}
		return strings.TrimSpace(envValue)
	}
	return trimmed
}

func resolveWorkspaceRoot(value string) string {
	defaultRoot := filepath.Join(os.TempDir(), "symphony_workspaces")
	if strings.TrimSpace(value) == "" {
		return defaultRoot
	}

	if envName, ok := envReferenceName(value); ok {
		envValue, exists := os.LookupEnv(envName)
		if !exists || strings.TrimSpace(envValue) == "" {
			return defaultRoot
		}
		value = envValue
	}

	return expandHomeDir(value)
}

func expandHomeDir(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return path
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return path
		}
		return filepath.Join(home, path[2:])
	}

	return path
}

func envReferenceName(value string) (string, bool) {
	if !strings.HasPrefix(value, "$") {
		return "", false
	}

	envName := strings.TrimPrefix(value, "$")
	if envName == "" {
		return "", false
	}

	for i, r := range envName {
		if i == 0 {
			if !isEnvNameStart(r) {
				return "", false
			}
			continue
		}
		if !isEnvNameContinue(r) {
			return "", false
		}
	}

	return envName, true
}

func isEnvNameStart(r rune) bool {
	return r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

func isEnvNameContinue(r rune) bool {
	return isEnvNameStart(r) || ('0' <= r && r <= '9')
}

func normalizeSecretValue(value string) string {
	return strings.TrimSpace(value)
}

func normalizeOptionalMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}

	normalized, ok := normalizeConfigValue(value).(map[string]any)
	if !ok {
		return nil
	}
	return normalized
}

func normalizeStateLimitKeys(value map[string]int) map[string]int {
	if len(value) == 0 {
		return map[string]int{}
	}

	normalized := make(map[string]int, len(value))
	for state, limit := range value {
		normalized[strings.ToLower(strings.TrimSpace(state))] = limit
	}
	return normalized
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
