package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/observability"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
	ansiGray   = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiPurple = "\x1b[35m"
	ansiHome   = "\x1b[H"
	ansiClear  = "\x1b[2J"
)

type FixtureProvenanceEntry struct {
	Source  string `json:"source,omitempty"`
	Derived string `json:"derived,omitempty"`
}

func Render(view observability.DashboardView) string {
	switch view.Mode {
	case observability.DashboardModeOffline:
		return RenderOffline()
	case observability.DashboardModeUnavailable:
		return RenderUnavailable(view.UnavailableReason)
	}

	lines := []string{
		colorize("╭─ SYMPHONY STATUS", ansiBold),
		colorize("│ Agents: ", ansiBold) + colorize(fmt.Sprintf("%d", view.Header.AgentCount), ansiGreen) + colorize("/", ansiGray) + colorize(fmt.Sprintf("%d", view.Header.MaxAgents), ansiGray),
		colorize("│ Throughput: ", ansiBold) + colorize(fmt.Sprintf("%s tps", FormatCount(view.Header.ThroughputTPS)), ansiCyan),
		colorize("│ Runtime: ", ansiBold) + colorize(formatRuntime(view.Header.RuntimeSeconds), ansiPurple),
		colorize("│ Tokens: ", ansiBold) +
			colorize("in "+FormatCount(view.Header.Tokens.InputTokens), ansiYellow) +
			colorize(" | ", ansiGray) +
			colorize("out "+FormatCount(view.Header.Tokens.OutputTokens), ansiYellow) +
			colorize(" | ", ansiGray) +
			colorize("total "+FormatCount(view.Header.Tokens.TotalTokens), ansiYellow),
		colorize("│ Rate Limits: ", ansiBold) + formatRateLimits(view.Header.RateLimits),
		colorize("│ Project: ", ansiBold) + colorize(firstNonEmpty(view.Header.ProjectURL, "n/a"), colorForOptional(view.Header.ProjectURL)),
	}
	if strings.TrimSpace(view.Header.DashboardURL) != "" {
		lines = append(lines, colorize("│ Dashboard: ", ansiBold)+colorize(view.Header.DashboardURL, ansiCyan))
	}
	lines = append(lines,
		colorize("│ Next refresh: ", ansiBold)+colorize(firstNonEmpty(view.Header.NextRefresh, "n/a"), colorForOptional(view.Header.NextRefresh)),
		colorize("├─ Running", ansiBold),
		"│",
		"│   "+colorize("ID       STAGE          PID      AGE / TURN   TOKENS     SESSION        EVENT", ansiGray),
		"│   "+colorize("───────────────────────────────────────────────────────────────────────────────────────────────────────────────", ansiGray),
	)
	if len(view.Running) == 0 {
		lines = append(lines, "│  "+colorize("No active agents", ansiGray), "│")
	} else {
		for _, row := range view.Running {
			lines = append(lines, renderRunningRow(row))
		}
		lines = append(lines, "│")
	}
	lines = append(lines, colorize("├─ Backoff queue", ansiBold), "│")
	if len(view.Retrying) == 0 {
		lines = append(lines, "│  "+colorize("No queued retries", ansiGray))
	} else {
		for _, row := range view.Retrying {
			lines = append(lines, renderRetryRow(row))
		}
	}
	lines = append(lines, "╰─")
	return terminalPrefix() + strings.Join(lines, "\n")
}

func RenderOffline() string {
	return terminalPrefix() + strings.Join([]string{
		colorize("╭─ SYMPHONY STATUS", ansiBold),
		colorize("│ app_status=offline", ansiRed),
		"╰─",
	}, "\n")
}

func RenderUnavailable(reason string) string {
	reason = firstNonEmpty(reason, "Snapshot unavailable")
	return terminalPrefix() + strings.Join([]string{
		colorize("╭─ SYMPHONY STATUS", ansiBold),
		colorize("│ "+reason, ansiRed),
		"╰─",
	}, "\n")
}

func terminalPrefix() string {
	return ansiHome + ansiClear
}

func LoadFixtureProvenance(path string) (map[string]FixtureProvenanceEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var parsed map[string]FixtureProvenanceEntry
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func renderRunningRow(row observability.RunningRow) string {
	statusColor := ansiBlue
	switch {
	case strings.Contains(row.Event, "token"):
		statusColor = ansiYellow
	case strings.Contains(row.Event, "task") || strings.Contains(row.Event, "command"):
		statusColor = ansiGreen
	case strings.Contains(row.Event, "turn completed"):
		statusColor = ansiPurple
	}
	return "│ " +
		colorize("●", statusColor) + " " +
		colorize(cell(row.ID, 8, false), ansiCyan) + " " +
		colorize(cell(row.State, 14, false), statusColor) + " " +
		colorize(cell(firstNonEmpty(row.PID, "n/a"), 8, false), ansiYellow) + " " +
		colorize(cell(formatRuntimeTurns(row.RuntimeSeconds, row.TurnCount), 12, false), ansiPurple) + " " +
		colorize(cell(FormatCount(row.TotalTokens), 10, true), ansiYellow) + " " +
		colorize(cell(CompactSessionID(row.SessionID), 14, false), ansiCyan) + " " +
		colorize(cell(Truncate(row.Event, 140), 44, false), statusColor)
}

func renderRetryRow(row observability.RetryRow) string {
	errorText := ""
	if strings.TrimSpace(row.Error) != "" {
		errorText = " " + colorize("error="+Truncate(SanitizeInline(row.Error), 96), ansiDim)
	}
	return "│  " + colorize("↻", ansiYellow) + " " +
		colorize(firstNonEmpty(row.ID, "unknown"), ansiRed) + " " +
		colorize(fmt.Sprintf("attempt=%d", row.Attempt), ansiYellow) +
		colorize(" in ", ansiDim) +
		colorize(formatDue(row.DueIn), ansiCyan) +
		errorText
}

func formatRateLimits(rateLimits *domain.RateLimits) string {
	if rateLimits == nil {
		return colorize("unavailable", ansiGray)
	}
	limitID := firstNonEmpty(rateLimits.LimitID, "unknown")
	return colorize(limitID, ansiYellow) +
		colorize(" | ", ansiGray) +
		colorize("primary "+formatBucket(rateLimits.Primary), ansiCyan) +
		colorize(" | ", ansiGray) +
		colorize("secondary "+formatBucket(rateLimits.Secondary), ansiCyan) +
		colorize(" | ", ansiGray) +
		colorize(formatCredits(rateLimits.Credits), ansiGreen)
}

func formatBucket(bucket *domain.RateLimitBucket) string {
	if bucket == nil {
		return "n/a"
	}
	switch {
	case bucket.Remaining != nil && bucket.Limit != nil:
		base := FormatCount(*bucket.Remaining) + "/" + FormatCount(*bucket.Limit)
		if bucket.ResetInSeconds != nil {
			return base + " reset " + FormatCount(*bucket.ResetInSeconds) + "s"
		}
		return base
	case bucket.Remaining != nil:
		return "remaining " + FormatCount(*bucket.Remaining)
	case bucket.Limit != nil:
		return "limit " + FormatCount(*bucket.Limit)
	default:
		return "n/a"
	}
}

func formatCredits(credits *domain.RateLimitCredits) string {
	if credits == nil {
		return "credits n/a"
	}
	if credits.Unlimited != nil && *credits.Unlimited {
		return "credits unlimited"
	}
	if credits.HasCredits != nil && *credits.HasCredits {
		if credits.Balance != nil {
			return fmt.Sprintf("credits %.2f", *credits.Balance)
		}
		return "credits available"
	}
	return "credits none"
}

func cell(value string, width int, right bool) string {
	value = Truncate(SanitizeInline(value), width)
	if right {
		return fmt.Sprintf("%*s", width, value)
	}
	return fmt.Sprintf("%-*s", width, value)
}

func colorize(value, code string) string {
	return code + value + ansiReset
}

func colorForOptional(value string) string {
	if strings.TrimSpace(value) == "" || value == "n/a" {
		return ansiGray
	}
	return ansiCyan
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
