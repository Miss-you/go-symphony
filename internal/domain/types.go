package domain

import "time"

// WorkItem is the provider-neutral runtime item shared across the core.
type WorkItem struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	State       string
	Priority    *int
	BranchName  string
	URL         string
	AssigneeID  string
	Labels      []string
	BlockedBy   []Blocker
	Routable    *bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// Blocker captures the minimal identity and state needed for dispatch gating.
type Blocker struct {
	ID         string
	Identifier string
	State      string
}

// ActiveRun is the projection of a currently running work item.
type ActiveRun struct {
	ItemID           string
	ItemIdentifier   string
	State            string
	WorkerHost       string
	WorkspacePath    string
	SessionID        string
	TurnCount        int
	StartedAt        time.Time
	LastEventAt      *time.Time
	LastEventKind    RunEventKind
	LastEventMessage string
	CodexTotals      CodexTotals
}

// RetryEntry is the retry-queue projection for a work item.
type RetryEntry struct {
	ItemID         string
	ItemIdentifier string
	Attempt        int
	DueAt          time.Time
	LastError      string
	WorkerHost     string
	WorkspacePath  string
}

// PollingState is the observable state of the orchestrator poll loop.
type PollingState struct {
	Checking   bool
	NextPollAt *time.Time
	Interval   time.Duration
}

// Snapshot is the projection source for later observability surfaces.
type Snapshot struct {
	Running     []ActiveRun
	Retrying    []RetryEntry
	Polling     PollingState
	CodexTotals CodexTotals
	RateLimits  *RateLimits
}

// CodexTotals captures aggregate token usage surfaced by the runtime.
type CodexTotals struct {
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	SecondsRunning int
}

// RateLimits captures the latest Codex rate-limit state visible to the runtime.
type RateLimits struct {
	LimitID   string
	Primary   *RateLimitBucket
	Secondary *RateLimitBucket
	Credits   *RateLimitCredits
}

// RateLimitBucket is one quota bucket in the Codex rate-limit payload.
type RateLimitBucket struct {
	Remaining      *int
	Limit          *int
	ResetInSeconds *int
}

// RateLimitCredits is the credit-related portion of the Codex rate-limit payload.
type RateLimitCredits struct {
	HasCredits *bool
	Unlimited  *bool
	Balance    *float64
}

// RunEventKind is the stable worker-to-orchestrator event vocabulary.
type RunEventKind string

const (
	RunEventWorkspaceCreated        RunEventKind = "workspace_created"
	RunEventWorkspacePathDiscovered RunEventKind = "workspace_path_discovered"
	RunEventRunnerHostSelected      RunEventKind = "runner_host_selected"
	RunEventCodexEventReceived      RunEventKind = "codex_event_received"
	RunEventTurnCompleted           RunEventKind = "turn_completed"
	RunEventRunCompleted            RunEventKind = "run_completed"
	RunEventRunFailed               RunEventKind = "run_failed"
	RunEventRetryScheduled          RunEventKind = "retry_scheduled"
)

// RunEvent is the tagged worker-reporting envelope consumed by the orchestrator.
type RunEvent struct {
	Kind           RunEventKind
	ItemID         string
	ItemIdentifier string
	At             time.Time
	Attempt        int
	WorkerHost     string
	WorkspacePath  string
	SessionID      string
	Message        string
	Err            error
	CodexTotals    CodexTotals
	RateLimits     *RateLimits
}
