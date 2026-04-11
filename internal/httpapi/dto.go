package httpapi

import (
	"path/filepath"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
)

type errorPayload struct {
	Error errorBody `json:"error"`
}

type stateErrorPayload struct {
	GeneratedAt string    `json:"generated_at"`
	Error       errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type statePayload struct {
	GeneratedAt string                `json:"generated_at"`
	Counts      countsPayload         `json:"counts"`
	Running     []runningEntryPayload `json:"running"`
	Retrying    []retryEntryPayload   `json:"retrying"`
	CodexTotals codexTotalsPayload    `json:"codex_totals"`
	RateLimits  *rateLimitsPayload    `json:"rate_limits"`
}

type countsPayload struct {
	Running  int `json:"running"`
	Retrying int `json:"retrying"`
}

type tokensPayload struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type codexTotalsPayload struct {
	InputTokens    int `json:"input_tokens"`
	OutputTokens   int `json:"output_tokens"`
	TotalTokens    int `json:"total_tokens"`
	SecondsRunning int `json:"seconds_running"`
}

type runningEntryPayload struct {
	IssueID         string        `json:"issue_id"`
	IssueIdentifier string        `json:"issue_identifier"`
	State           string        `json:"state"`
	SessionID       string        `json:"session_id"`
	TurnCount       int           `json:"turn_count"`
	LastEvent       string        `json:"last_event"`
	LastMessage     string        `json:"last_message"`
	StartedAt       *string       `json:"started_at"`
	LastEventAt     *string       `json:"last_event_at"`
	Tokens          tokensPayload `json:"tokens"`
}

type retryEntryPayload struct {
	IssueID         string  `json:"issue_id"`
	IssueIdentifier string  `json:"issue_identifier"`
	Attempt         int     `json:"attempt"`
	DueAt           *string `json:"due_at"`
	Error           string  `json:"error"`
}

type rateLimitsPayload struct {
	LimitID   string                   `json:"limit_id,omitempty"`
	Primary   *rateLimitBucketPayload  `json:"primary,omitempty"`
	Secondary *rateLimitBucketPayload  `json:"secondary,omitempty"`
	Credits   *rateLimitCreditsPayload `json:"credits,omitempty"`
}

type rateLimitBucketPayload struct {
	Remaining      *int `json:"remaining,omitempty"`
	Limit          *int `json:"limit,omitempty"`
	ResetInSeconds *int `json:"reset_in_seconds,omitempty"`
}

type rateLimitCreditsPayload struct {
	HasCredits *bool    `json:"has_credits,omitempty"`
	Unlimited  *bool    `json:"unlimited,omitempty"`
	Balance    *float64 `json:"balance,omitempty"`
}

type issuePayload struct {
	IssueIdentifier string                 `json:"issue_identifier"`
	IssueID         string                 `json:"issue_id"`
	Status          string                 `json:"status"`
	Workspace       workspacePayload       `json:"workspace"`
	Attempts        attemptsPayload        `json:"attempts"`
	Running         *runningIssuePayload   `json:"running"`
	Retry           *retryIssuePayload     `json:"retry"`
	Logs            logsPayload            `json:"logs"`
	RecentEvents    []recentEventPayload   `json:"recent_events"`
	LastError       *string                `json:"last_error"`
	Tracked         map[string]interface{} `json:"tracked"`
}

type workspacePayload struct {
	Path string `json:"path"`
}

type attemptsPayload struct {
	RestartCount        int `json:"restart_count"`
	CurrentRetryAttempt int `json:"current_retry_attempt"`
}

type runningIssuePayload struct {
	SessionID   string        `json:"session_id"`
	TurnCount   int           `json:"turn_count"`
	State       string        `json:"state"`
	StartedAt   *string       `json:"started_at"`
	LastEvent   string        `json:"last_event"`
	LastMessage string        `json:"last_message"`
	LastEventAt *string       `json:"last_event_at"`
	Tokens      tokensPayload `json:"tokens"`
}

type retryIssuePayload struct {
	Attempt int     `json:"attempt"`
	DueAt   *string `json:"due_at"`
	Error   string  `json:"error"`
}

type logsPayload struct {
	CodexSessionLogs []interface{} `json:"codex_session_logs"`
}

type recentEventPayload struct {
	At      string `json:"at"`
	Event   string `json:"event"`
	Message string `json:"message"`
}

type refreshPayload struct {
	Queued      bool     `json:"queued"`
	Coalesced   bool     `json:"coalesced"`
	RequestedAt string   `json:"requested_at"`
	Operations  []string `json:"operations"`
}

func newStatePayload(snapshot domain.Snapshot, generatedAt string) statePayload {
	running := make([]runningEntryPayload, 0, len(snapshot.Running))
	for _, entry := range snapshot.Running {
		running = append(running, newRunningEntryPayload(entry))
	}
	retrying := make([]retryEntryPayload, 0, len(snapshot.Retrying))
	for _, entry := range snapshot.Retrying {
		retrying = append(retrying, newRetryEntryPayload(entry))
	}
	return statePayload{
		GeneratedAt: generatedAt,
		Counts: countsPayload{
			Running:  len(snapshot.Running),
			Retrying: len(snapshot.Retrying),
		},
		Running:  running,
		Retrying: retrying,
		CodexTotals: codexTotalsPayload{
			InputTokens:    snapshot.CodexTotals.InputTokens,
			OutputTokens:   snapshot.CodexTotals.OutputTokens,
			TotalTokens:    snapshot.CodexTotals.TotalTokens,
			SecondsRunning: snapshot.CodexTotals.SecondsRunning,
		},
		RateLimits: newRateLimitsPayload(snapshot.RateLimits),
	}
}

func newRunningEntryPayload(entry domain.ActiveRun) runningEntryPayload {
	return runningEntryPayload{
		IssueID:         entry.ItemID,
		IssueIdentifier: entry.ItemIdentifier,
		State:           entry.State,
		SessionID:       entry.SessionID,
		TurnCount:       entry.TurnCount,
		LastEvent:       string(entry.LastEventKind),
		LastMessage:     entry.LastEventMessage,
		StartedAt:       formatTime(entry.StartedAt),
		LastEventAt:     formatOptionalTime(entry.LastEventAt),
		Tokens:          newTokensPayload(entry.CodexTotals),
	}
}

func newRetryEntryPayload(entry domain.RetryEntry) retryEntryPayload {
	return retryEntryPayload{
		IssueID:         entry.ItemID,
		IssueIdentifier: entry.ItemIdentifier,
		Attempt:         entry.Attempt,
		DueAt:           formatTime(entry.DueAt),
		Error:           entry.LastError,
	}
}

func newIssuePayload(snapshot domain.Snapshot, identifier, workspaceRoot string) (issuePayload, bool) {
	running := findRunning(snapshot.Running, identifier)
	retry := findRetry(snapshot.Retrying, identifier)
	if running == nil && retry == nil {
		return issuePayload{}, false
	}

	payload := issuePayload{
		IssueIdentifier: identifier,
		IssueID:         issueID(running, retry),
		Status:          issueStatus(running, retry),
		Workspace:       workspacePayload{Path: issueWorkspacePath(running, retry, workspaceRoot, identifier)},
		Attempts:        attemptsPayload{RestartCount: restartCount(retry), CurrentRetryAttempt: retryAttempt(retry)},
		Logs:            logsPayload{CodexSessionLogs: []interface{}{}},
		RecentEvents:    []recentEventPayload{},
		Tracked:         map[string]interface{}{},
	}
	if running != nil {
		payload.Running = &runningIssuePayload{
			SessionID:   running.SessionID,
			TurnCount:   running.TurnCount,
			State:       running.State,
			StartedAt:   formatTime(running.StartedAt),
			LastEvent:   string(running.LastEventKind),
			LastMessage: running.LastEventMessage,
			LastEventAt: formatOptionalTime(running.LastEventAt),
			Tokens:      newTokensPayload(running.CodexTotals),
		}
		if running.LastEventAt != nil {
			payload.RecentEvents = append(payload.RecentEvents, recentEventPayload{
				At:      formatTimeValue(*running.LastEventAt),
				Event:   string(running.LastEventKind),
				Message: running.LastEventMessage,
			})
		}
	}
	if retry != nil {
		payload.Retry = &retryIssuePayload{
			Attempt: retry.Attempt,
			DueAt:   formatTime(retry.DueAt),
			Error:   retry.LastError,
		}
		payload.LastError = stringPtr(retry.LastError)
	}
	return payload, true
}

func newTokensPayload(totals domain.CodexTotals) tokensPayload {
	return tokensPayload{
		InputTokens:  totals.InputTokens,
		OutputTokens: totals.OutputTokens,
		TotalTokens:  totals.TotalTokens,
	}
}

func newRateLimitsPayload(rateLimits *domain.RateLimits) *rateLimitsPayload {
	if rateLimits == nil {
		return nil
	}
	return &rateLimitsPayload{
		LimitID:   rateLimits.LimitID,
		Primary:   newRateLimitBucketPayload(rateLimits.Primary),
		Secondary: newRateLimitBucketPayload(rateLimits.Secondary),
		Credits:   newRateLimitCreditsPayload(rateLimits.Credits),
	}
}

func newRateLimitBucketPayload(bucket *domain.RateLimitBucket) *rateLimitBucketPayload {
	if bucket == nil {
		return nil
	}
	return &rateLimitBucketPayload{
		Remaining:      bucket.Remaining,
		Limit:          bucket.Limit,
		ResetInSeconds: bucket.ResetInSeconds,
	}
}

func newRateLimitCreditsPayload(credits *domain.RateLimitCredits) *rateLimitCreditsPayload {
	if credits == nil {
		return nil
	}
	return &rateLimitCreditsPayload{
		HasCredits: credits.HasCredits,
		Unlimited:  credits.Unlimited,
		Balance:    credits.Balance,
	}
}

func findRunning(entries []domain.ActiveRun, identifier string) *domain.ActiveRun {
	for i := range entries {
		if entries[i].ItemIdentifier == identifier {
			return &entries[i]
		}
	}
	return nil
}

func findRetry(entries []domain.RetryEntry, identifier string) *domain.RetryEntry {
	for i := range entries {
		if entries[i].ItemIdentifier == identifier {
			return &entries[i]
		}
	}
	return nil
}

func issueID(running *domain.ActiveRun, retry *domain.RetryEntry) string {
	if running != nil {
		return running.ItemID
	}
	return retry.ItemID
}

func issueStatus(running *domain.ActiveRun, _ *domain.RetryEntry) string {
	if running != nil {
		return "running"
	}
	return "retrying"
}

func issueWorkspacePath(running *domain.ActiveRun, retry *domain.RetryEntry, root, identifier string) string {
	if running != nil && running.WorkspacePath != "" {
		return running.WorkspacePath
	}
	if retry != nil && retry.WorkspacePath != "" {
		return retry.WorkspacePath
	}
	return filepath.Join(root, identifier)
}

func retryAttempt(retry *domain.RetryEntry) int {
	if retry == nil {
		return 0
	}
	return retry.Attempt
}

func restartCount(retry *domain.RetryEntry) int {
	attempt := retryAttempt(retry)
	if attempt <= 0 {
		return 0
	}
	return attempt - 1
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func formatTime(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	formatted := formatTimeValue(t)
	return &formatted
}

func formatTimeValue(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func stringPtr(value string) *string {
	return &value
}
