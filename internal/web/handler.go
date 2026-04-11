package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	termui "github.com/Miss-you/go-symphony/internal/dashboard"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/httpapi"
	"github.com/Miss-you/go-symphony/internal/observability"
)

const assetCacheControl = "public, max-age=31536000"

//go:embed assets/dashboard.css assets/vendor/phoenix_html/phoenix_html.js assets/vendor/phoenix/phoenix.js assets/vendor/phoenix_live_view/phoenix_live_view.js
var assetFS embed.FS

type Options struct {
	Snapshot      httpapi.SnapshotFunc
	Refresh       httpapi.RefreshFunc
	WorkspaceRoot string
	Now           func() time.Time
	MaxAgents     int
	DashboardURL  string
	ProjectURL    string
}

type handler struct {
	snapshot     httpapi.SnapshotFunc
	now          func() time.Time
	maxAgents    int
	dashboardURL string
	projectURL   string
	api          http.Handler

	mu        sync.Mutex
	projector *observability.Projector
}

type assetRoute struct {
	file        string
	contentType string
}

var assetRoutes = map[string]assetRoute{
	"/dashboard.css":                                 {file: "assets/dashboard.css", contentType: "text/css; charset=utf-8"},
	"/vendor/phoenix_html/phoenix_html.js":           {file: "assets/vendor/phoenix_html/phoenix_html.js", contentType: "application/javascript; charset=utf-8"},
	"/vendor/phoenix/phoenix.js":                     {file: "assets/vendor/phoenix/phoenix.js", contentType: "application/javascript; charset=utf-8"},
	"/vendor/phoenix_live_view/phoenix_live_view.js": {file: "assets/vendor/phoenix_live_view/phoenix_live_view.js", contentType: "application/javascript; charset=utf-8"},
}

func NewHandler(opts Options) http.Handler {
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &handler{
		snapshot:     opts.Snapshot,
		now:          now,
		maxAgents:    opts.MaxAgents,
		dashboardURL: opts.DashboardURL,
		projectURL:   opts.ProjectURL,
		api: httpapi.NewHandler(httpapi.Options{
			Snapshot:      opts.Snapshot,
			Refresh:       opts.Refresh,
			WorkspaceRoot: opts.WorkspaceRoot,
			Now:           now,
		}),
		projector: observability.NewProjector(),
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if route, ok := assetRoutes[r.URL.Path]; ok {
		h.serveAsset(w, r, route)
		return
	}
	if isAssetPath(r.URL.Path) {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	if isAPIRoute(r.URL.Path) {
		h.api.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		h.serveRoot(w, r)
		return
	}
	writeJSONError(w, http.StatusNotFound, "not_found", "Route not found")
}

func (h *handler) serveAsset(w http.ResponseWriter, r *http.Request, route assetRoute) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	body, err := assetFS.ReadFile(route.file)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", route.contentType)
	w.Header().Set("Cache-Control", assetCacheControl)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *handler) serveRoot(w http.ResponseWriter, r *http.Request) {
	view := h.view(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = pageTemplate.Execute(w, pageData{View: view, RateLimits: prettyRateLimits(view.Header.RateLimits)})
}

func (h *handler) view(ctx context.Context) observability.DashboardView {
	if h.snapshot == nil {
		return h.projector.Unavailable("snapshot_unavailable: Snapshot unavailable")
	}
	snapshot, err := h.snapshot(ctx)
	if err != nil {
		return h.projector.Unavailable(snapshotErrorReason(err))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.projector.Project(snapshot, observability.DashboardContext{
		Now:          h.now(),
		MaxAgents:    h.maxAgents,
		DashboardURL: h.dashboardURL,
		ProjectURL:   h.projectURL,
	})
}

func snapshotErrorReason(err error) string {
	if errors.Is(err, httpapi.ErrSnapshotTimeout) {
		return "snapshot_timeout: Snapshot timed out"
	}
	return "snapshot_unavailable: Snapshot unavailable"
}

func isAPIRoute(path string) bool {
	return path == "/api/v1" || strings.HasPrefix(path, "/api/v1/")
}

func isAssetPath(path string) bool {
	return path == "/dashboard.css" || strings.HasPrefix(path, "/vendor/")
}

type pageData struct {
	View       observability.DashboardView
	RateLimits string
}

func prettyRateLimits(rateLimits *domain.RateLimits) string {
	if rateLimits == nil {
		return "n/a"
	}
	body, err := json.MarshalIndent(rateLimits, "", "  ")
	if err != nil {
		return "n/a"
	}
	return string(body)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

var pageTemplate = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"count":      termui.FormatCount,
	"runtime":    formatRuntime,
	"runtimeRow": formatRuntimeTurns,
	"session":    termui.CompactSessionID,
	"inline":     termui.SanitizeInline,
	"truncate":   termui.Truncate,
}).Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Symphony Observability</title>
    <link rel="stylesheet" href="/dashboard.css">
    <script defer src="/vendor/phoenix_html/phoenix_html.js"></script>
    <script defer src="/vendor/phoenix/phoenix.js"></script>
    <script defer src="/vendor/phoenix_live_view/phoenix_live_view.js"></script>
  </head>
  <body>
    <main class="app-shell">
      <section class="dashboard-shell">
        <header class="hero-card">
          <div class="hero-grid">
            <div>
              <p class="eyebrow">Symphony Observability</p>
              <h1 class="hero-title">Operations Dashboard</h1>
              <p class="hero-copy">Current state, retry pressure, token usage, and orchestration health for the active Symphony runtime.</p>
            </div>
            <div class="status-stack" aria-label="Connection status">
              <span class="status-badge status-badge-live"><span class="status-badge-dot"></span>Live</span>
              <span class="status-badge status-badge-offline"><span class="status-badge-dot"></span>Offline</span>
            </div>
          </div>
        </header>

        {{if eq .View.Mode "unavailable"}}
          <section class="error-card">
            <h2 class="error-title">Snapshot unavailable</h2>
            <p class="error-copy">{{.View.UnavailableReason}}</p>
          </section>
        {{else}}
          <section class="metric-grid" aria-label="Runtime metrics">
            <article class="metric-card">
              <p class="metric-label">Running</p>
              <p class="metric-value numeric">{{count .View.Header.AgentCount}}</p>
              <p class="metric-detail">Active issue sessions in the current runtime.</p>
            </article>
            <article class="metric-card">
              <p class="metric-label">Retrying</p>
              <p class="metric-value numeric">{{count (len .View.Retrying)}}</p>
              <p class="metric-detail">Issues waiting for the next retry window.</p>
            </article>
            <article class="metric-card">
              <p class="metric-label">Total tokens</p>
              <p class="metric-value numeric">{{count .View.Header.Tokens.TotalTokens}}</p>
              <p class="metric-detail numeric">In {{count .View.Header.Tokens.InputTokens}} / Out {{count .View.Header.Tokens.OutputTokens}}</p>
            </article>
            <article class="metric-card">
              <p class="metric-label">Runtime</p>
              <p class="metric-value numeric">{{runtime .View.Header.RuntimeSeconds}}</p>
              <p class="metric-detail">Total Codex runtime across completed and active sessions.</p>
            </article>
          </section>

          <section class="section-card">
            <div class="section-header">
              <div>
                <h2 class="section-title">Rate limits</h2>
                <p class="section-copy">Latest upstream rate-limit snapshot, when available.</p>
              </div>
            </div>
            <pre class="code-panel">{{.RateLimits}}</pre>
          </section>

          <section class="section-card">
            <div class="section-header">
              <div>
                <h2 class="section-title">Running sessions</h2>
                <p class="section-copy">Active issues, last known agent activity, and token usage.</p>
              </div>
            </div>
            {{if .View.Running}}
              <div class="table-wrap">
                <table class="data-table data-table-running">
                  <thead>
                    <tr>
                      <th>Issue</th>
                      <th>State</th>
                      <th>Session</th>
                      <th>Runtime / turns</th>
                      <th>Codex update</th>
                      <th>Tokens</th>
                    </tr>
                  </thead>
                  <tbody>
                    {{range .View.Running}}
                      <tr>
                        <td>
                          <div class="issue-stack">
                            <span class="issue-id">{{.ID}}</span>
                            <a class="issue-link" href="/api/v1/{{.ID}}">JSON details</a>
                          </div>
                        </td>
                        <td><span class="state-badge">{{.State}}</span></td>
                        <td>
                          {{if .SessionID}}
                            <button type="button" class="subtle-button" data-copy="{{.SessionID}}" onclick="navigator.clipboard && navigator.clipboard.writeText(this.dataset.copy); this.textContent = 'Copied';">Copy ID</button>
                            <span class="muted mono">{{session .SessionID}}</span>
                          {{else}}
                            <span class="muted">n/a</span>
                          {{end}}
                        </td>
                        <td class="numeric">{{runtimeRow .RuntimeSeconds .TurnCount}}</td>
                        <td><span class="event-text">{{truncate (inline .Event) 120}}</span></td>
                        <td class="numeric">{{count .TotalTokens}}</td>
                      </tr>
                    {{end}}
                  </tbody>
                </table>
              </div>
            {{else}}
              <p class="empty-state">No active sessions.</p>
            {{end}}
          </section>

          <section class="section-card">
            <div class="section-header">
              <div>
                <h2 class="section-title">Retry queue</h2>
                <p class="section-copy">Issues waiting for the next retry window.</p>
              </div>
            </div>
            {{if .View.Retrying}}
              <div class="table-wrap">
                <table class="data-table">
                  <thead>
                    <tr>
                      <th>Issue</th>
                      <th>Attempt</th>
                      <th>Due in</th>
                      <th>Error</th>
                    </tr>
                  </thead>
                  <tbody>
                    {{range .View.Retrying}}
                      <tr>
                        <td>
                          <div class="issue-stack">
                            <span class="issue-id">{{.ID}}</span>
                            <a class="issue-link" href="/api/v1/{{.ID}}">JSON details</a>
                          </div>
                        </td>
                        <td class="numeric">{{.Attempt}}</td>
                        <td class="mono numeric">{{printf "%.0fs" .DueIn.Seconds}}</td>
                        <td>{{truncate (inline .Error) 120}}</td>
                      </tr>
                    {{end}}
                  </tbody>
                </table>
              </div>
            {{else}}
              <p class="empty-state">No issues are currently backing off.</p>
            {{end}}
          </section>
        {{end}}
      </section>
    </main>
  </body>
</html>`))

func formatRuntime(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
}

func formatRuntimeTurns(seconds, turns int) string {
	base := formatRuntime(seconds)
	if turns <= 0 {
		return base
	}
	return base + " / " + termui.FormatCount(turns)
}
