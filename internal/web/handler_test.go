package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/httpapi"
)

func TestHandlerRootRendersDashboardFromSnapshot(t *testing.T) {
	now := time.Date(2026, 4, 12, 2, 0, 0, 0, time.UTC)
	started := now.Add(-90 * time.Second)
	due := now.Add(30 * time.Second)
	remaining := 42
	limit := 100

	handler := NewHandler(Options{
		Now:          fixedNow(now),
		MaxAgents:    10,
		DashboardURL: "http://127.0.0.1:4000/",
		ProjectURL:   "https://linear.app/project/project/issues",
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			return domain.Snapshot{
				Running: []domain.ActiveRun{{
					ItemID:           "issue-web",
					ItemIdentifier:   "MT-WEB",
					State:            "In Progress",
					WorkerHost:       "local",
					SessionID:        "thread-web-123456",
					TurnCount:        3,
					StartedAt:        started,
					LastEventKind:    domain.RunEventCodexEventReceived,
					LastEventMessage: "turn completed (completed)",
					CodexTotals:      domain.CodexTotals{InputTokens: 50, OutputTokens: 70, TotalTokens: 120},
				}},
				Retrying: []domain.RetryEntry{{
					ItemID:         "issue-retry",
					ItemIdentifier: "MT-RETRY",
					Attempt:        2,
					DueAt:          due,
					LastError:      "rate limit",
				}},
				CodexTotals: domain.CodexTotals{InputTokens: 1000, OutputTokens: 2000, TotalTokens: 3000, SecondsRunning: 360},
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

	rec := perform(handler, http.MethodGet, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want text/html; charset=utf-8", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<title>Symphony Observability</title>",
		"Operations Dashboard",
		"Live",
		"Offline",
		"Running",
		"Retrying",
		"Total tokens",
		"Runtime",
		"Rate limits",
		"Running sessions",
		"Retry queue",
		"MT-WEB",
		"Copy ID",
		`href="/api/v1/MT-WEB"`,
		"JSON details",
		"turn completed (completed)",
		"MT-RETRY",
		"rate limit",
		"3,000",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("root body missing %q:\n%s", want, body)
		}
	}
}

func TestHandlerRootReflectsLaterSnapshots(t *testing.T) {
	now := time.Date(2026, 4, 12, 2, 0, 0, 0, time.UTC)
	calls := 0
	handler := NewHandler(Options{
		Now: fixedNow(now),
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			calls++
			if calls == 1 {
				return domain.Snapshot{}, nil
			}
			return domain.Snapshot{Running: []domain.ActiveRun{{ItemID: "issue-new", ItemIdentifier: "MT-NEW", State: "Todo"}}}, nil
		},
	})

	first := perform(handler, http.MethodGet, "/").Body.String()
	if !strings.Contains(first, "No active sessions.") || strings.Contains(first, "MT-NEW") {
		t.Fatalf("first render did not use empty snapshot:\n%s", first)
	}
	second := perform(handler, http.MethodGet, "/").Body.String()
	if !strings.Contains(second, "MT-NEW") {
		t.Fatalf("second render did not reflect later snapshot:\n%s", second)
	}
}

func TestHandlerUnavailableAndRouteErrors(t *testing.T) {
	handler := NewHandler(Options{
		Now:      fixedNow(time.Date(2026, 4, 12, 2, 0, 0, 0, time.UTC)),
		Snapshot: func(context.Context) (domain.Snapshot, error) { return domain.Snapshot{}, httpapi.ErrSnapshotTimeout },
	})

	unavailable := perform(handler, http.MethodGet, "/")
	if unavailable.Code != http.StatusOK {
		t.Fatalf("unavailable status = %d, want %d", unavailable.Code, http.StatusOK)
	}
	if body := unavailable.Body.String(); !strings.Contains(body, "Snapshot unavailable") || !strings.Contains(body, "snapshot_timeout") {
		t.Fatalf("unavailable body missing error panel:\n%s", body)
	}

	assertJSONError(t, perform(handler, http.MethodPost, "/"), http.StatusMethodNotAllowed, "method_not_allowed")
	assertJSONError(t, perform(handler, http.MethodGet, "/unknown"), http.StatusNotFound, "not_found")
}

func TestHandlerStaticAssets(t *testing.T) {
	handler := NewHandler(Options{})

	css := perform(handler, http.MethodGet, "/dashboard.css")
	if css.Code != http.StatusOK {
		t.Fatalf("css status = %d, want %d", css.Code, http.StatusOK)
	}
	if got := css.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Fatalf("css content-type = %q", got)
	}
	if got := css.Header().Get("Cache-Control"); got != "public, max-age=31536000" {
		t.Fatalf("css cache-control = %q", got)
	}
	if !strings.Contains(css.Body.String(), ".dashboard-shell") {
		t.Fatalf("css body missing dashboard shell rule")
	}

	for _, path := range []string{
		"/vendor/phoenix_html/phoenix_html.js",
		"/vendor/phoenix/phoenix.js",
		"/vendor/phoenix_live_view/phoenix_live_view.js",
	} {
		vendor := perform(handler, http.MethodGet, path)
		if vendor.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, vendor.Code, http.StatusOK)
		}
		if got := vendor.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
			t.Fatalf("%s content-type = %q", path, got)
		}
		if got := vendor.Header().Get("Cache-Control"); got != "public, max-age=31536000" {
			t.Fatalf("%s cache-control = %q", path, got)
		}
		if strings.TrimSpace(vendor.Body.String()) == "" {
			t.Fatalf("%s body is empty", path)
		}
	}

	missing := perform(handler, http.MethodGet, "/vendor/phoenix/missing.js")
	if missing.Code != http.StatusNotFound || strings.TrimSpace(missing.Body.String()) != "Not Found" {
		t.Fatalf("missing asset = %d %q, want 404 Not Found", missing.Code, missing.Body.String())
	}
}

func TestHandlerDelegatesAPIRoutes(t *testing.T) {
	now := time.Date(2026, 4, 12, 2, 0, 0, 0, time.UTC)
	handler := NewHandler(Options{
		Now: fixedNow(now),
		Snapshot: func(context.Context) (domain.Snapshot, error) {
			return domain.Snapshot{Running: []domain.ActiveRun{{ItemID: "issue-api", ItemIdentifier: "MT-API"}}}, nil
		},
		Refresh: func(context.Context) (httpapi.RefreshResult, error) {
			return httpapi.RefreshResult{Queued: true}, nil
		},
	})

	state := decodeObject(t, perform(handler, http.MethodGet, "/api/v1/state"))
	counts := state["counts"].(map[string]any)
	if counts["running"] != float64(1) {
		t.Fatalf("delegated state running = %#v, want 1", counts["running"])
	}

	refresh := decodeObject(t, perform(handler, http.MethodPost, "/api/v1/refresh"))
	if refresh["queued"] != true {
		t.Fatalf("delegated refresh queued = %#v, want true", refresh["queued"])
	}
}

func TestWebPackageDoesNotImportRuntimeOwnersOrProviders(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, output)
	}
	deps := strings.Split(strings.TrimSpace(string(output)), "\n")
	forbidden := []string{
		"github.com/Miss-you/go-symphony/internal/orchestrator",
		"github.com/Miss-you/go-symphony/internal/trackers/linear",
		"github.com/Miss-you/go-symphony/internal/workflow",
	}
	for _, forbiddenImport := range forbidden {
		for _, dep := range deps {
			if dep == forbiddenImport {
				t.Fatalf("forbidden dependency %q found in deps:\n%s", forbiddenImport, output)
			}
		}
	}
}

func perform(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeObject(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, rec.Body.String())
	}
	return got
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	got := decodeObject(t, rec)
	errBody, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error body = %#v", got["error"])
	}
	if errBody["code"] != code {
		t.Fatalf("error code = %#v, want %q", errBody["code"], code)
	}
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}
