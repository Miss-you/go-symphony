package observability

import (
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/domain"
)

func TestProjectorBuildsDashboardViewAndThroughput(t *testing.T) {
	base := time.Date(2026, 4, 12, 1, 0, 0, 0, time.UTC)
	projector := NewProjector()

	first := projector.Project(domain.Snapshot{
		CodexTotals: domain.CodexTotals{InputTokens: 100, OutputTokens: 50, TotalTokens: 150, SecondsRunning: 60},
		Polling:     domain.PollingState{Interval: 5 * time.Second},
	}, DashboardContext{Now: base, MaxAgents: 10, ProjectURL: "https://linear.app/project/project/issues"})
	if first.Header.AgentCount != 0 || first.Header.MaxAgents != 10 {
		t.Fatalf("agents = %d/%d, want 0/10", first.Header.AgentCount, first.Header.MaxAgents)
	}
	if first.Header.NextRefresh != "n/a" {
		t.Fatalf("next refresh = %q, want n/a", first.Header.NextRefresh)
	}
	if first.Header.ThroughputTPS != 0 {
		t.Fatalf("initial tps = %d, want 0", first.Header.ThroughputTPS)
	}

	second := projector.Project(domain.Snapshot{
		Running: []domain.ActiveRun{{
			ItemID:           "item-1",
			ItemIdentifier:   "MT-1",
			State:            "running",
			WorkerHost:       "local",
			SessionID:        "thread-1234567890",
			TurnCount:        2,
			StartedAt:        base.Add(-90 * time.Second),
			LastEventKind:    domain.RunEventCodexEventReceived,
			LastEventMessage: "turn completed (completed)",
			CodexTotals:      domain.CodexTotals{TotalTokens: 200},
		}},
		Retrying: []domain.RetryEntry{{
			ItemID:         "item-2",
			ItemIdentifier: "MT-2",
			Attempt:        3,
			DueAt:          base.Add(2 * time.Second),
			LastError:      "rate limit\nretry",
		}},
		CodexTotals: domain.CodexTotals{InputTokens: 200, OutputTokens: 100, TotalTokens: 300, SecondsRunning: 90},
		Polling:     domain.PollingState{NextPollAt: ptrTime(base.Add(2 * time.Second)), Interval: 5 * time.Second},
		RateLimits:  &domain.RateLimits{LimitID: "gpt-5"},
	}, DashboardContext{Now: base.Add(5 * time.Second), MaxAgents: 10, ProjectURL: "https://linear.app/project/project/issues", DashboardURL: "http://127.0.0.1:4000/"})

	if second.Header.AgentCount != 1 {
		t.Fatalf("agent count = %d, want 1", second.Header.AgentCount)
	}
	if second.Header.ThroughputTPS != 30 {
		t.Fatalf("tps = %d, want 30", second.Header.ThroughputTPS)
	}
	if second.Header.NextRefresh != "checking now..." {
		t.Fatalf("next refresh = %q, want checking now...", second.Header.NextRefresh)
	}
	if got := second.Running[0].Event; got != "turn completed (completed)" {
		t.Fatalf("running event = %q", got)
	}
	if got := second.Retrying[0].Error; got != "rate limit retry" {
		t.Fatalf("retry error = %q", got)
	}

	throttled := projector.Project(domain.Snapshot{
		CodexTotals: domain.CodexTotals{TotalTokens: 900},
	}, DashboardContext{Now: base.Add(5*time.Second + 100*time.Millisecond)})
	if throttled.Header.ThroughputTPS != second.Header.ThroughputTPS {
		t.Fatalf("same-second tps = %d, want %d", throttled.Header.ThroughputTPS, second.Header.ThroughputTPS)
	}
}

func TestProjectorModes(t *testing.T) {
	projector := NewProjector()
	if got := projector.Offline().Mode; got != DashboardModeOffline {
		t.Fatalf("offline mode = %q", got)
	}
	unavailable := projector.Unavailable("Orchestrator snapshot unavailable")
	if unavailable.Mode != DashboardModeUnavailable || unavailable.UnavailableReason != "Orchestrator snapshot unavailable" {
		t.Fatalf("unavailable view = %#v", unavailable)
	}
}

func TestSummarizeCodexEvent(t *testing.T) {
	tests := []struct {
		name  string
		event codex.Event
		want  string
	}{
		{
			name:  "turn completed usage",
			event: codex.Event{Kind: codex.EventTurnCompleted, Method: "turn/completed", Payload: map[string]any{"params": map[string]any{"usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}}},
			want:  "turn completed (completed) (in 10, out 5, total 15)",
		},
		{
			name:  "exec command wrapper",
			event: codex.Event{Kind: codex.EventUnknownMessage, Method: "codex/event/exec_command_begin", Payload: map[string]any{"params": map[string]any{"msg": map[string]any{"command": "go test ./..."}}}},
			want:  "go test ./...",
		},
		{
			name:  "agent message delta",
			event: codex.Event{Kind: codex.EventUnknownMessage, Method: "codex/event/agent_message_delta", Payload: map[string]any{"params": map[string]any{"msg": map[string]any{"payload": map[string]any{"delta": "working on it"}}}}},
			want:  "agent message streaming: working on it",
		},
		{
			name:  "malformed",
			event: codex.Event{Kind: codex.EventMalformedMessage},
			want:  "malformed JSON event from codex",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeCodexEvent(tt.event); got != tt.want {
				t.Fatalf("summary = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
