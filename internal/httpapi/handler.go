package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
)

var (
	ErrSnapshotTimeout     = errors.New("snapshot timeout")
	ErrSnapshotUnavailable = errors.New("snapshot unavailable")
	ErrRefreshUnavailable  = errors.New("refresh unavailable")
)

type SnapshotFunc func(context.Context) (domain.Snapshot, error)

type RefreshFunc func(context.Context) (RefreshResult, error)

type RefreshResult struct {
	Queued    bool
	Coalesced bool
}

type Options struct {
	Snapshot      SnapshotFunc
	Refresh       RefreshFunc
	WorkspaceRoot string
	Now           func() time.Time
}

type handler struct {
	snapshot      SnapshotFunc
	refresh       RefreshFunc
	workspaceRoot string
	now           func() time.Time
}

func NewHandler(opts Options) http.Handler {
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &handler{
		snapshot:      opts.Snapshot,
		refresh:       opts.Refresh,
		workspaceRoot: opts.WorkspaceRoot,
		now:           now,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/state":
		if r.Method != http.MethodGet {
			h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		h.handleState(w, r)
	case "/api/v1/refresh":
		if r.Method != http.MethodPost {
			h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		h.handleRefresh(w, r)
	default:
		identifier, ok := issueIdentifier(r.URL.Path)
		if !ok {
			h.writeError(w, http.StatusNotFound, "not_found", "Route not found")
			return
		}
		if r.Method != http.MethodGet {
			h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		h.handleIssue(w, r, identifier)
	}
}

func (h *handler) handleState(w http.ResponseWriter, r *http.Request) {
	generatedAt := formatTimeValue(h.now())
	snapshot, err := h.loadSnapshot(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrSnapshotTimeout):
			h.writeJSON(w, http.StatusOK, stateErrorPayload{
				GeneratedAt: generatedAt,
				Error:       errorBody{Code: "snapshot_timeout", Message: "Snapshot timed out"},
			})
		default:
			h.writeJSON(w, http.StatusOK, stateErrorPayload{
				GeneratedAt: generatedAt,
				Error:       errorBody{Code: "snapshot_unavailable", Message: "Snapshot unavailable"},
			})
		}
		return
	}
	h.writeJSON(w, http.StatusOK, newStatePayload(snapshot, generatedAt))
}

func (h *handler) handleIssue(w http.ResponseWriter, r *http.Request, identifier string) {
	snapshot, err := h.loadSnapshot(r.Context())
	if err != nil {
		h.writeError(w, http.StatusNotFound, "issue_not_found", "Issue not found")
		return
	}
	payload, ok := newIssuePayload(snapshot, identifier, h.workspaceRoot)
	if !ok {
		h.writeError(w, http.StatusNotFound, "issue_not_found", "Issue not found")
		return
	}
	h.writeJSON(w, http.StatusOK, payload)
}

func (h *handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if h.refresh == nil {
		h.writeError(w, http.StatusServiceUnavailable, "orchestrator_unavailable", "Orchestrator is unavailable")
		return
	}
	result, err := h.refresh(r.Context())
	if err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "orchestrator_unavailable", "Orchestrator is unavailable")
		return
	}
	h.writeJSON(w, http.StatusAccepted, refreshPayload{
		Queued:      result.Queued,
		Coalesced:   result.Coalesced,
		RequestedAt: formatTimeValue(h.now()),
		Operations:  []string{"poll", "reconcile"},
	})
}

func (h *handler) loadSnapshot(ctx context.Context) (domain.Snapshot, error) {
	if h.snapshot == nil {
		return domain.Snapshot{}, ErrSnapshotUnavailable
	}
	return h.snapshot(ctx)
}

func (h *handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, errorPayload{Error: errorBody{Code: code, Message: message}})
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func issueIdentifier(path string) (string, bool) {
	const prefix = "/api/v1/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	identifier := strings.TrimPrefix(path, prefix)
	if identifier == "" || strings.Contains(identifier, "/") {
		return "", false
	}
	return identifier, true
}
