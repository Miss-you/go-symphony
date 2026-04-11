package observability

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/domain"
)

type DashboardMode string

const (
	DashboardModeNormal      DashboardMode = "normal"
	DashboardModeOffline     DashboardMode = "offline"
	DashboardModeUnavailable DashboardMode = "unavailable"
)

type DashboardContext struct {
	Now          time.Time
	MaxAgents    int
	DashboardURL string
	ProjectURL   string
}

type DashboardHeader struct {
	AgentCount     int
	MaxAgents      int
	ThroughputTPS  int
	RuntimeSeconds int
	Tokens         domain.CodexTotals
	RateLimits     *domain.RateLimits
	ProjectURL     string
	DashboardURL   string
	NextRefresh    string
}

type DashboardView struct {
	Mode              DashboardMode
	UnavailableReason string
	Header            DashboardHeader
	Running           []RunningRow
	Retrying          []RetryRow
}

type RunningRow struct {
	ID             string
	State          string
	PID            string
	RuntimeSeconds int
	TurnCount      int
	TotalTokens    int
	SessionID      string
	Event          string
}

type RetryRow struct {
	ID      string
	Attempt int
	DueIn   time.Duration
	Error   string
}

type Projector struct {
	samples       []tokenSample
	lastTPSSecond int64
	lastTPSValue  int
	hasLastTPS    bool
}

type tokenSample struct {
	at     time.Time
	tokens int
}

func NewProjector() *Projector {
	return &Projector{}
}

func (p *Projector) Project(snapshot domain.Snapshot, ctx DashboardContext) DashboardView {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	running := make([]RunningRow, 0, len(snapshot.Running))
	for _, entry := range snapshot.Running {
		pid := strings.TrimSpace(entry.WorkerHost)
		if pid == "" {
			pid = "n/a"
		}
		runtimeSeconds := entry.CodexTotals.SecondsRunning
		if runtimeSeconds <= 0 && !entry.StartedAt.IsZero() {
			runtimeSeconds = max(0, int(now.Sub(entry.StartedAt)/time.Second))
		}
		running = append(running, RunningRow{
			ID:             firstNonEmpty(entry.ItemIdentifier, entry.ItemID, "unknown"),
			State:          firstNonEmpty(entry.State, "unknown"),
			PID:            pid,
			RuntimeSeconds: runtimeSeconds,
			TurnCount:      entry.TurnCount,
			TotalTokens:    entry.CodexTotals.TotalTokens,
			SessionID:      entry.SessionID,
			Event:          firstNonEmpty(entry.LastEventMessage, string(entry.LastEventKind), "no codex message yet"),
		})
	}
	retrying := make([]RetryRow, 0, len(snapshot.Retrying))
	for _, entry := range snapshot.Retrying {
		dueIn := entry.DueAt.Sub(now)
		if dueIn < 0 {
			dueIn = 0
		}
		retrying = append(retrying, RetryRow{
			ID:      firstNonEmpty(entry.ItemIdentifier, entry.ItemID, "unknown"),
			Attempt: entry.Attempt,
			DueIn:   dueIn,
			Error:   sanitizeInline(entry.LastError),
		})
	}
	sort.SliceStable(retrying, func(i, j int) bool {
		if retrying[i].DueIn != retrying[j].DueIn {
			return retrying[i].DueIn < retrying[j].DueIn
		}
		return retrying[i].ID < retrying[j].ID
	})
	return DashboardView{
		Mode: DashboardModeNormal,
		Header: DashboardHeader{
			AgentCount:     len(running),
			MaxAgents:      ctx.MaxAgents,
			ThroughputTPS:  p.throughput(now, snapshot.CodexTotals.TotalTokens),
			RuntimeSeconds: snapshot.CodexTotals.SecondsRunning,
			Tokens:         snapshot.CodexTotals,
			RateLimits:     snapshot.RateLimits,
			ProjectURL:     ctx.ProjectURL,
			DashboardURL:   ctx.DashboardURL,
			NextRefresh:    nextRefresh(snapshot.Polling, now),
		},
		Running:  running,
		Retrying: retrying,
	}
}

func (p *Projector) Offline() DashboardView {
	return DashboardView{Mode: DashboardModeOffline}
}

func (p *Projector) Unavailable(reason string) DashboardView {
	return DashboardView{Mode: DashboardModeUnavailable, UnavailableReason: firstNonEmpty(reason, "Snapshot unavailable")}
}

func (p *Projector) throughput(now time.Time, total int) int {
	p.samples = append(p.samples, tokenSample{at: now, tokens: total})
	cutoff := now.Add(-5 * time.Second)
	kept := p.samples[:0]
	for _, sample := range p.samples {
		if !sample.at.Before(cutoff) {
			kept = append(kept, sample)
		}
	}
	p.samples = kept
	second := now.Unix()
	if p.hasLastTPS && p.lastTPSSecond == second {
		return p.lastTPSValue
	}
	if len(p.samples) < 2 {
		p.lastTPSSecond = second
		p.lastTPSValue = 0
		p.hasLastTPS = true
		return 0
	}
	first := p.samples[0]
	elapsed := now.Sub(first.at).Seconds()
	value := 0
	if elapsed > 0 {
		value = int(math.Trunc(float64(max(0, total-first.tokens)) / elapsed))
	}
	p.lastTPSSecond = second
	p.lastTPSValue = value
	p.hasLastTPS = true
	return value
}

func nextRefresh(polling domain.PollingState, now time.Time) string {
	if polling.Checking {
		return "checking now..."
	}
	if polling.NextPollAt == nil {
		return "n/a"
	}
	due := polling.NextPollAt.Sub(now)
	if due <= 0 {
		return "checking now..."
	}
	seconds := int(math.Ceil(due.Seconds()))
	return fmt.Sprintf("%ds", seconds)
}

func SummarizeCodexEvent(event codex.Event) string {
	switch event.Kind {
	case codex.EventSessionStarted:
		id := firstNonEmpty(event.SessionID, event.ThreadID)
		if id != "" {
			return "session started (" + id + ")"
		}
		return "session started"
	case codex.EventMalformedMessage:
		return "malformed JSON event from codex"
	case codex.EventTurnFailed:
		return firstNonEmpty(humanizeMethod("turn/failed", event.Payload), "turn failed")
	case codex.EventTurnCancelled:
		return "turn cancelled"
	case codex.EventApprovalAnswered:
		return appendIfValue(firstNonEmpty(humanizeMethod(event.Method, event.Payload), "approval request auto-approved"), "auto-approved")
	case codex.EventToolInputAnswered:
		return firstNonEmpty(humanizeMethod("item/tool/requestUserInput", event.Payload), "tool input auto-answered")
	}
	if event.Method != "" {
		return firstNonEmpty(humanizeMethod(event.Method, event.Payload), event.Method)
	}
	return firstNonEmpty(event.Message, string(event.Kind))
}

func humanizeMethod(method string, payload map[string]any) string {
	switch method {
	case "turn/started":
		if id := stringAt(payload, "params", "turn", "id"); id != "" {
			return "turn started (" + id + ")"
		}
		return "turn started"
	case "turn/completed":
		status := firstNonEmpty(stringAt(payload, "params", "turn", "status"), "completed")
		base := "turn completed (" + status + ")"
		if usage := formatUsage(firstMapAt(payload, []string{"params", "usage"}, []string{"params", "tokenUsage"})); usage != "" {
			return base + " (" + usage + ")"
		}
		return base
	case "turn/failed":
		if msg := stringAt(payload, "params", "error", "message"); msg != "" {
			return "turn failed: " + msg
		}
		return "turn failed"
	case "turn/cancelled":
		return "turn cancelled"
	case "turn/diff/updated":
		return "turn diff updated"
	case "turn/plan/updated":
		return "plan updated"
	case "thread/tokenUsage/updated":
		if usage := formatUsage(firstMapAt(payload, []string{"params", "tokenUsage", "total"})); usage != "" {
			return "thread token usage updated (" + usage + ")"
		}
		return "thread token usage updated"
	case "item/started":
		return itemLifecycle("started", payload)
	case "item/completed":
		return itemLifecycle("completed", payload)
	case "item/tool/requestUserInput", "tool/requestUserInput":
		if question := firstNonEmpty(stringAt(payload, "params", "question"), stringAt(payload, "params", "prompt")); question != "" {
			return "tool requires user input: " + truncate(sanitizeInline(question), 80)
		}
		return "tool requires user input"
	case "item/commandExecution/requestApproval":
		if command := normalizeCommand(anyAt(payload, "params", "command")); command != "" {
			return "command approval requested (" + command + ")"
		}
		return "command approval requested"
	}
	if strings.HasPrefix(method, "codex/event/") {
		return humanizeWrapper(strings.TrimPrefix(method, "codex/event/"), payload)
	}
	return ""
}

func humanizeWrapper(name string, payload map[string]any) string {
	switch name {
	case "exec_command_begin":
		if command := firstNonEmpty(normalizeCommand(anyAt(payload, "params", "msg", "command")), normalizeCommand(anyAt(payload, "params", "msg", "parsed_cmd"))); command != "" {
			return command
		}
		return "command started"
	case "exec_command_end":
		if code, ok := intAt(payload, "params", "msg", "exit_code"); ok {
			return fmt.Sprintf("command completed (exit %d)", code)
		}
		return "command completed"
	case "agent_message_delta":
		return streaming("agent message streaming", payload)
	case "agent_reasoning":
		return streaming("reasoning update", payload)
	case "agent_reasoning_delta":
		return streaming("reasoning streaming", payload)
	case "token_count":
		if usage := formatUsage(firstMapAt(payload, []string{"params", "msg", "payload", "info", "total_token_usage"}, []string{"params", "tokenUsage", "total"})); usage != "" {
			return "token count update (" + usage + ")"
		}
		return "token count update"
	case "task_started":
		return "task started"
	case "mcp_startup_complete":
		return "mcp startup complete"
	}
	return strings.ReplaceAll(name, "_", " ")
}

func streaming(label string, payload map[string]any) string {
	for _, path := range [][]string{
		{"params", "delta"},
		{"params", "msg", "delta"},
		{"params", "msg", "payload", "delta"},
		{"params", "msg", "payload", "text"},
		{"params", "msg", "payload", "content"},
	} {
		if value := stringAt(payload, path...); value != "" {
			return label + ": " + truncate(sanitizeInline(value), 80)
		}
	}
	return label
}

func itemLifecycle(state string, payload map[string]any) string {
	itemType := firstNonEmpty(stringAt(payload, "params", "item", "type"), "item")
	itemType = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(itemType, "_", " ")))
	return "item " + state + ": " + itemType
}

func formatUsage(usage map[string]any) string {
	if usage == nil {
		return ""
	}
	var parts []string
	if value, ok := intFromKeys(usage, "input_tokens", "inputTokens", "prompt_tokens", "promptTokens"); ok {
		parts = append(parts, fmt.Sprintf("in %d", value))
	}
	if value, ok := intFromKeys(usage, "output_tokens", "outputTokens", "completion_tokens", "completionTokens"); ok {
		parts = append(parts, fmt.Sprintf("out %d", value))
	}
	if value, ok := intFromKeys(usage, "total_tokens", "totalTokens", "total"); ok {
		parts = append(parts, fmt.Sprintf("total %d", value))
	}
	return strings.Join(parts, ", ")
}

func appendIfValue(base, suffix string) string {
	if strings.Contains(base, suffix) {
		return base
	}
	return base + " (" + suffix + ")"
}

func firstMapAt(root map[string]any, paths ...[]string) map[string]any {
	for _, path := range paths {
		if value, ok := anyAt(root, path...).(map[string]any); ok {
			return value
		}
	}
	return nil
}

func anyAt(root map[string]any, path ...string) any {
	var current any = root
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	return current
}

func stringAt(root map[string]any, path ...string) string {
	value, _ := anyAt(root, path...).(string)
	return value
}

func intAt(root map[string]any, path ...string) (int, bool) {
	return asInt(anyAt(root, path...))
}

func intFromKeys(root map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		if value, ok := asInt(root[key]); ok {
			return value, true
		}
	}
	return 0, false
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func normalizeCommand(value any) string {
	switch v := value.(type) {
	case string:
		return truncate(sanitizeInline(v), 80)
	case []string:
		return truncate(sanitizeInline(strings.Join(v, " ")), 80)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			part, ok := item.(string)
			if !ok {
				return ""
			}
			parts = append(parts, part)
		}
		return truncate(sanitizeInline(strings.Join(parts, " ")), 80)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sanitizeInline(value string) string {
	replacer := strings.NewReplacer("\\r\\n", " ", "\\r", " ", "\\n", " ", "\r\n", " ", "\r", " ", "\n", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}
