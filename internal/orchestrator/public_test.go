package orchestrator_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/orchestrator"
)

func TestServiceCanBeDrivenThroughExportedRuntimeSeam(t *testing.T) {
	settings := config.Settings{
		Provider: config.ProviderSettings{
			ActiveStates:   []string{"Todo", "In Progress"},
			TerminalStates: []string{"Done"},
		},
		Polling: config.PollingSettings{IntervalMS: 30_000},
		Agent: config.AgentSettings{
			MaxConcurrentAgents:        1,
			MaxRetryBackoffMS:          300_000,
			MaxConcurrentAgentsByState: map[string]int{},
		},
		Codex: config.CodexSettings{StallTimeoutMS: 300_000},
	}

	started := make(chan struct{})
	var startOnce sync.Once
	svc := orchestrator.Start(settings, orchestrator.Dependencies{
		ListCandidates: func(context.Context) ([]domain.WorkItem, error) {
			return []domain.WorkItem{{
				ID:         "item-1",
				Identifier: "MT-1",
				Title:      "Do work",
				State:      "In Progress",
			}}, nil
		},
		RefreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			if len(ids) != 1 || ids[0] != "item-1" {
				t.Fatalf("refresh ids = %v, want [item-1]", ids)
			}
			return []domain.WorkItem{{
				ID:         "item-1",
				Identifier: "MT-1",
				Title:      "Do work",
				State:      "In Progress",
			}}, nil
		},
		StartRun: func(context.Context, orchestrator.StartRunRequest) (orchestrator.StartRunResult, error) {
			startOnce.Do(func() { close(started) })
			return orchestrator.StartRunResult{WorkerHost: "", Handle: "handle"}, nil
		},
	})
	defer func() { _ = svc.Close() }()

	svc.RequestRefresh()
	svc.ApplyRunEvent(domain.RunEvent{
		Kind:           domain.RunEventCodexEventReceived,
		ItemID:         "missing-before-dispatch",
		ItemIdentifier: "MT-X",
		At:             time.Now(),
		Message:        "ignored",
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("StartRun was not called through exported runtime seam")
	}

	snapshot := svc.Snapshot()
	if snapshot.Polling.Interval != 30*time.Second {
		t.Fatalf("poll interval = %s, want 30s", snapshot.Polling.Interval)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}
