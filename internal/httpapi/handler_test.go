package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
)

func TestHandlerStatePayload(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 999, time.UTC)
	started := time.Date(2026, 4, 12, 0, 1, 2, 0, time.UTC)
	lastEvent := time.Date(2026, 4, 12, 0, 2, 3, 0, time.UTC)
	dueAt := time.Date(2026, 4, 12, 0, 3, 4, 0, time.UTC)
	remaining := 11
	limit := 20

	handler := NewHandler(Options{
		Now: fixedNow(now),
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			return domain.Snapshot{
				Running: []domain.ActiveRun{{
					ItemID:           "issue-http",
					ItemIdentifier:   "MT-HTTP",
					State:            "In Progress",
					SessionID:        "thread-http",
					TurnCount:        7,
					StartedAt:        started,
					LastEventAt:      &lastEvent,
					LastEventKind:    domain.RunEventCodexEventReceived,
					LastEventMessage: "working",
					CodexTotals:      domain.CodexTotals{InputTokens: 4, OutputTokens: 8, TotalTokens: 12},
				}},
				Retrying: []domain.RetryEntry{{
					ItemID:         "issue-retry",
					ItemIdentifier: "MT-RETRY",
					Attempt:        2,
					DueAt:          dueAt,
					LastError:      "boom",
				}},
				CodexTotals: domain.CodexTotals{InputTokens: 40, OutputTokens: 80, TotalTokens: 120, SecondsRunning: 42},
				RateLimits: &domain.RateLimits{
					LimitID: "codex",
					Primary: &domain.RateLimitBucket{
						Remaining: &remaining,
						Limit:     &limit,
					},
				},
			}, nil
		},
	})

	rec := perform(handler, http.MethodGet, "/api/v1/state")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeObject(t, rec)

	want := map[string]any{
		"generated_at": "2026-04-12T00:00:00Z",
		"counts": map[string]any{
			"running":  float64(1),
			"retrying": float64(1),
		},
		"running": []any{map[string]any{
			"issue_id":         "issue-http",
			"issue_identifier": "MT-HTTP",
			"state":            "In Progress",
			"session_id":       "thread-http",
			"turn_count":       float64(7),
			"last_event":       "codex_event_received",
			"last_message":     "working",
			"started_at":       "2026-04-12T00:01:02Z",
			"last_event_at":    "2026-04-12T00:02:03Z",
			"tokens": map[string]any{
				"input_tokens":  float64(4),
				"output_tokens": float64(8),
				"total_tokens":  float64(12),
			},
		}},
		"retrying": []any{map[string]any{
			"issue_id":         "issue-retry",
			"issue_identifier": "MT-RETRY",
			"attempt":          float64(2),
			"due_at":           "2026-04-12T00:03:04Z",
			"error":            "boom",
		}},
		"codex_totals": map[string]any{
			"input_tokens":    float64(40),
			"output_tokens":   float64(80),
			"total_tokens":    float64(120),
			"seconds_running": float64(42),
		},
		"rate_limits": map[string]any{
			"limit_id": "codex",
			"primary": map[string]any{
				"remaining": float64(11),
				"limit":     float64(20),
			},
		},
	}
	assertDeepEqual(t, got, want)
}

func TestHandlerStatePayloadEmptyAndSnapshotErrors(t *testing.T) {
	handler := NewHandler(Options{
		Now: fixedNow(time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)),
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			return domain.Snapshot{}, nil
		},
	})

	got := decodeObject(t, perform(handler, http.MethodGet, "/api/v1/state"))
	assertDeepEqual(t, got["running"], []any{})
	assertDeepEqual(t, got["retrying"], []any{})
	if got["rate_limits"] != nil {
		t.Fatalf("rate_limits = %#v, want nil", got["rate_limits"])
	}

	timeoutHandler := NewHandler(Options{
		Now:      fixedNow(time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)),
		Snapshot: func(context.Context) (domain.Snapshot, error) { return domain.Snapshot{}, wrap(ErrSnapshotTimeout) },
	})
	assertResponse(t, timeoutHandler, http.MethodGet, "/api/v1/state", http.StatusOK, map[string]any{
		"generated_at": "2026-04-12T00:00:00Z",
		"error":        map[string]any{"code": "snapshot_timeout", "message": "Snapshot timed out"},
	})

	unavailableHandler := NewHandler(Options{Now: fixedNow(time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC))})
	assertResponse(t, unavailableHandler, http.MethodGet, "/api/v1/state", http.StatusOK, map[string]any{
		"generated_at": "2026-04-12T00:00:00Z",
		"error":        map[string]any{"code": "snapshot_unavailable", "message": "Snapshot unavailable"},
	})
}

func TestHandlerIssuePayloads(t *testing.T) {
	lastEvent := time.Date(2026, 4, 12, 0, 2, 3, 0, time.UTC)
	dueAt := time.Date(2026, 4, 12, 0, 3, 4, 0, time.UTC)
	handler := NewHandler(Options{
		WorkspaceRoot: "/tmp/symphony_workspaces",
		Now:           fixedNow(time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)),
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			return domain.Snapshot{
				Running: []domain.ActiveRun{
					{
						ItemID:           "issue-http",
						ItemIdentifier:   "MT-HTTP",
						State:            "In Progress",
						WorkspacePath:    "/actual/MT-HTTP",
						SessionID:        "thread-http",
						TurnCount:        7,
						LastEventAt:      &lastEvent,
						LastEventKind:    domain.RunEventCodexEventReceived,
						LastEventMessage: "working",
						CodexTotals:      domain.CodexTotals{InputTokens: 4, OutputTokens: 8, TotalTokens: 12},
					},
					{
						ItemID:         "issue-both-run",
						ItemIdentifier: "MT-BOTH",
						WorkspacePath:  "/actual/MT-BOTH",
					},
				},
				Retrying: []domain.RetryEntry{
					{ItemID: "issue-retry", ItemIdentifier: "MT-RETRY", Attempt: 2, DueAt: dueAt, LastError: "boom"},
					{ItemID: "issue-both-retry", ItemIdentifier: "MT-BOTH", Attempt: 3, DueAt: dueAt, LastError: "later"},
				},
			}, nil
		},
	})

	running := decodeObject(t, perform(handler, http.MethodGet, "/api/v1/MT-HTTP"))
	assertDeepEqual(t, running["status"], "running")
	assertDeepEqual(t, running["issue_id"], "issue-http")
	assertDeepEqual(t, running["workspace"], map[string]any{"path": "/actual/MT-HTTP"})
	assertDeepEqual(t, running["attempts"], map[string]any{"restart_count": float64(0), "current_retry_attempt": float64(0)})
	assertDeepEqual(t, running["logs"], map[string]any{"codex_session_logs": []any{}})
	assertDeepEqual(t, running["tracked"], map[string]any{})
	assertDeepEqual(t, running["recent_events"], []any{map[string]any{
		"at":      "2026-04-12T00:02:03Z",
		"event":   "codex_event_received",
		"message": "working",
	}})
	if running["retry"] != nil {
		t.Fatalf("running retry = %#v, want nil", running["retry"])
	}

	retry := decodeObject(t, perform(handler, http.MethodGet, "/api/v1/MT-RETRY"))
	assertDeepEqual(t, retry["status"], "retrying")
	assertDeepEqual(t, retry["issue_id"], "issue-retry")
	assertDeepEqual(t, retry["workspace"], map[string]any{"path": "/tmp/symphony_workspaces/MT-RETRY"})
	assertDeepEqual(t, retry["attempts"], map[string]any{"restart_count": float64(1), "current_retry_attempt": float64(2)})
	assertDeepEqual(t, retry["retry"], map[string]any{"attempt": float64(2), "due_at": "2026-04-12T00:03:04Z", "error": "boom"})
	assertDeepEqual(t, retry["last_error"], "boom")
	if retry["running"] != nil {
		t.Fatalf("retry running = %#v, want nil", retry["running"])
	}

	both := decodeObject(t, perform(handler, http.MethodGet, "/api/v1/MT-BOTH"))
	assertDeepEqual(t, both["status"], "running")
	assertDeepEqual(t, both["issue_id"], "issue-both-run")
	if both["running"] == nil || both["retry"] == nil {
		t.Fatalf("both running/retry = %#v/%#v, want both present", both["running"], both["retry"])
	}

	assertResponse(t, handler, http.MethodGet, "/api/v1/MT-MISSING", http.StatusNotFound, map[string]any{
		"error": map[string]any{"code": "issue_not_found", "message": "Issue not found"},
	})
}

func TestHandlerRefreshPayloads(t *testing.T) {
	handler := NewHandler(Options{
		Now: fixedNow(time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)),
		Refresh: func(context.Context) (RefreshResult, error) {
			return RefreshResult{Queued: true, Coalesced: false}, nil
		},
	})

	assertResponse(t, handler, http.MethodPost, "/api/v1/refresh", http.StatusAccepted, map[string]any{
		"queued":       true,
		"coalesced":    false,
		"requested_at": "2026-04-12T00:00:00Z",
		"operations":   []any{"poll", "reconcile"},
	})

	unavailableHandler := NewHandler(Options{
		Refresh: func(context.Context) (RefreshResult, error) {
			return RefreshResult{}, wrap(ErrRefreshUnavailable)
		},
	})
	assertResponse(t, unavailableHandler, http.MethodPost, "/api/v1/refresh", http.StatusServiceUnavailable, map[string]any{
		"error": map[string]any{"code": "orchestrator_unavailable", "message": "Orchestrator is unavailable"},
	})

	nilRefreshHandler := NewHandler(Options{})
	assertResponse(t, nilRefreshHandler, http.MethodPost, "/api/v1/refresh", http.StatusServiceUnavailable, map[string]any{
		"error": map[string]any{"code": "orchestrator_unavailable", "message": "Orchestrator is unavailable"},
	})
}

func TestHandlerRouteErrors(t *testing.T) {
	handler := NewHandler(Options{})

	cases := []struct {
		name   string
		method string
		path   string
		status int
		body   map[string]any
	}{
		{
			name:   "state method not allowed",
			method: http.MethodPost,
			path:   "/api/v1/state",
			status: http.StatusMethodNotAllowed,
			body:   map[string]any{"error": map[string]any{"code": "method_not_allowed", "message": "Method not allowed"}},
		},
		{
			name:   "refresh method not allowed",
			method: http.MethodGet,
			path:   "/api/v1/refresh",
			status: http.StatusMethodNotAllowed,
			body:   map[string]any{"error": map[string]any{"code": "method_not_allowed", "message": "Method not allowed"}},
		},
		{
			name:   "issue method not allowed",
			method: http.MethodPost,
			path:   "/api/v1/MT-1",
			status: http.StatusMethodNotAllowed,
			body:   map[string]any{"error": map[string]any{"code": "method_not_allowed", "message": "Method not allowed"}},
		},
		{
			name:   "unknown path",
			method: http.MethodGet,
			path:   "/unknown",
			status: http.StatusNotFound,
			body:   map[string]any{"error": map[string]any{"code": "not_found", "message": "Route not found"}},
		},
		{
			name:   "nested api path is not issue identifier",
			method: http.MethodGet,
			path:   "/api/v1/MT-1/extra",
			status: http.StatusNotFound,
			body:   map[string]any{"error": map[string]any{"code": "not_found", "message": "Route not found"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertResponse(t, handler, tc.method, tc.path, tc.status, tc.body)
		})
	}
}

func TestHTTPAPIPackageDependencyBoundary(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", ".")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps . failed: %v", err)
	}

	forbidden := []string{
		"github.com/Miss-you/go-symphony/internal/orchestrator",
		"github.com/Miss-you/go-symphony/internal/cli",
		"github.com/Miss-you/go-symphony/internal/tracker",
		"github.com/Miss-you/go-symphony/internal/trackers",
		"github.com/Miss-you/go-symphony/internal/toolbridge",
	}
	deps := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, dep := range deps {
		for _, forbiddenDep := range forbidden {
			if dep == forbiddenDep || strings.HasPrefix(dep, forbiddenDep+"/") {
				t.Fatalf("internal/httpapi dependency graph includes forbidden dependency %q", dep)
			}
		}
	}
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func wrap(err error) error {
	return errors.Join(errors.New("wrapped"), err)
}

func perform(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertResponse(t *testing.T, handler http.Handler, method, path string, wantStatus int, wantBody map[string]any) {
	t.Helper()
	rec := perform(handler, method, path)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	assertDeepEqual(t, decodeObject(t, rec), wantBody)
}

func decodeObject(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return got
}

func assertDeepEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}
