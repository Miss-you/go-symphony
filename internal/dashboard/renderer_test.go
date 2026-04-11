package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/observability"
)

func TestRenderSnapshotFixtures(t *testing.T) {
	for _, name := range []string{
		"idle",
		"idle_with_dashboard_url",
		"super_busy",
		"backoff_queue",
		"credits_unlimited",
		"snapshot_unavailable",
		"orchestrator_snapshot_unavailable",
		"offline",
	} {
		t.Run(name, func(t *testing.T) {
			got := escapeANSI(Render(testView(name)))
			want := readFixture(t, name+".snapshot.txt")
			if got != strings.TrimSuffix(want, "\n") {
				t.Fatalf("fixture %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

func TestFixtureProvenance(t *testing.T) {
	provenance := loadFixtureProvenance(t, filepath.Join("testdata", "status_dashboard_snapshots", "provenance.json"))
	fixtures := goFixtureNames(t)
	for _, fixture := range fixtures {
		entry, ok := provenance[fixture]
		if !ok {
			t.Fatalf("missing provenance for %s", fixture)
		}
		if entry.Source == "" {
			if strings.TrimSpace(entry.Derived) == "" {
				t.Fatalf("derived fixture %s has no reason", fixture)
			}
			continue
		}
		sourcePath := filepath.Join("testdata", "status_dashboard_snapshots", "source", entry.Source)
		if _, err := os.Stat(sourcePath); err != nil {
			t.Fatalf("missing source fixture %s: %v", sourcePath, err)
		}
		source := readPath(t, sourcePath)
		goFixture := readFixture(t, fixture)
		if sourceSkeleton, goSkeleton := normalizedSkeleton(source), normalizedSkeleton(goFixture); sourceSkeleton != goSkeleton {
			t.Fatalf("normalized skeleton mismatch for %s\nsource:\n%s\n\ngo:\n%s", fixture, sourceSkeleton, goSkeleton)
		}
		if sourceRows, goRows := runningIDs(source), runningIDs(goFixture); strings.Join(sourceRows, ",") != strings.Join(goRows, ",") {
			t.Fatalf("running row ids mismatch for %s: source=%v go=%v", fixture, sourceRows, goRows)
		}
		if sourceRetries, goRetries := retryRows(source), retryRows(goFixture); strings.Join(sourceRetries, "\n") != strings.Join(goRetries, "\n") {
			t.Fatalf("retry rows mismatch for %s\nsource:\n%s\n\ngo:\n%s", fixture, strings.Join(sourceRetries, "\n"), strings.Join(goRetries, "\n"))
		}
	}
	for fixture := range provenance {
		if !contains(fixtures, fixture) {
			t.Fatalf("provenance references missing Go fixture %s", fixture)
		}
	}
}

func goFixtureNames(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("testdata", "status_dashboard_snapshots", "*.snapshot.txt"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, filepath.Base(match))
	}
	sort.Strings(names)
	return names
}

func loadFixtureProvenance(t *testing.T, path string) map[string]FixtureProvenanceEntry {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var provenance map[string]FixtureProvenanceEntry
	if err := json.Unmarshal(content, &provenance); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return provenance
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	return readPath(t, filepath.Join("testdata", "status_dashboard_snapshots", name))
}

func readPath(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func escapeANSI(value string) string {
	return strings.ReplaceAll(value, "\x1b", `\e`)
}

func normalizedSkeleton(value string) string {
	re := regexp.MustCompile(`\\e\[[0-9;]*[A-Za-z]`)
	value = re.ReplaceAllString(value, "")
	lines := strings.Split(strings.TrimSpace(value), "\n")
	var kept []string
	for _, line := range lines {
		if strings.Contains(line, "SYMPHONY STATUS") ||
			strings.Contains(line, "Agents:") ||
			strings.Contains(line, "Throughput:") ||
			strings.Contains(line, "Runtime:") ||
			strings.Contains(line, "Tokens:") ||
			strings.Contains(line, "Rate Limits:") ||
			strings.Contains(line, "Project:") ||
			strings.Contains(line, "Dashboard:") ||
			strings.Contains(line, "Next refresh:") ||
			strings.Contains(line, "├─ Running") ||
			strings.Contains(line, "├─ Backoff queue") ||
			strings.Contains(line, "No active agents") ||
			strings.Contains(line, "No queued retries") ||
			strings.Contains(line, "↻") ||
			strings.Contains(line, "╰─") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func runningIDs(value string) []string {
	return identifiersFromLines(value, "●")
}

func retryRows(value string) []string {
	rows := identifiersFromLines(value, "↻")
	for i, row := range rows {
		rows[i] = strings.Join(strings.Fields(stripANSI(row)), " ")
	}
	return rows
}

func identifiersFromLines(value string, marker string) []string {
	var ids []string
	re := regexp.MustCompile(`MT-[0-9A-Z]+`)
	for _, line := range strings.Split(value, "\n") {
		if !strings.Contains(line, marker) {
			continue
		}
		if id := re.FindString(line); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func stripANSI(value string) string {
	re := regexp.MustCompile(`\\e\[[0-9;]*[A-Za-z]`)
	return re.ReplaceAllString(value, "")
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func testView(name string) observability.DashboardView {
	primaryRemaining := 12345
	primaryLimit := 20000
	primaryReset := 30
	secondaryRemaining := 45
	secondaryLimit := 60
	secondaryReset := 12
	backoffPrimaryReset := 95
	backoffSecondaryReset := 45
	hasCredits := true
	balance := 9876.5
	unlimited := true
	noCredits := false

	base := observability.DashboardView{
		Mode: observability.DashboardModeNormal,
		Header: observability.DashboardHeader{
			MaxAgents:      10,
			ProjectURL:     "https://linear.app/project/project/issues",
			NextRefresh:    "n/a",
			RuntimeSeconds: 0,
		},
	}
	switch name {
	case "idle":
		return base
	case "idle_with_dashboard_url":
		base.Header.DashboardURL = "http://127.0.0.1:4000/"
		return base
	case "super_busy":
		base.Header.AgentCount = 2
		base.Header.ThroughputTPS = 1842
		base.Header.RuntimeSeconds = 4321
		base.Header.Tokens = domain.CodexTotals{InputTokens: 250000, OutputTokens: 18500, TotalTokens: 268500}
		base.Header.RateLimits = &domain.RateLimits{
			LimitID: "gpt-5",
			Primary: &domain.RateLimitBucket{
				Remaining:      &primaryRemaining,
				Limit:          &primaryLimit,
				ResetInSeconds: &primaryReset,
			},
			Secondary: &domain.RateLimitBucket{
				Remaining:      &secondaryRemaining,
				Limit:          &secondaryLimit,
				ResetInSeconds: &secondaryReset,
			},
			Credits: &domain.RateLimitCredits{HasCredits: &hasCredits, Balance: &balance},
		}
		base.Running = []observability.RunningRow{
			{ID: "MT-101", State: "running", PID: "4242", RuntimeSeconds: 785, TurnCount: 11, TotalTokens: 120450, SessionID: "thread-1234567890", Event: "turn completed (completed)"},
			{ID: "MT-102", State: "running", PID: "5252", RuntimeSeconds: 412, TurnCount: 4, TotalTokens: 89200, SessionID: "thread-abcdef1234567890", Event: "go test ./..."},
		}
		return base
	case "backoff_queue":
		base.Header.AgentCount = 1
		base.Header.ThroughputTPS = 15
		base.Header.RuntimeSeconds = 2700
		base.Header.Tokens = domain.CodexTotals{InputTokens: 18000, OutputTokens: 2200, TotalTokens: 20200}
		base.Header.RateLimits = &domain.RateLimits{
			LimitID:   "gpt-5",
			Primary:   &domain.RateLimitBucket{Remaining: &noInt, Limit: &primaryLimit, ResetInSeconds: &backoffPrimaryReset},
			Secondary: &domain.RateLimitBucket{Remaining: &noInt, Limit: &secondaryLimit, ResetInSeconds: &backoffSecondaryReset},
			Credits:   &domain.RateLimitCredits{HasCredits: &noCredits},
		}
		base.Running = []observability.RunningRow{{ID: "MT-638", State: "retrying", PID: "4242", RuntimeSeconds: 1225, TurnCount: 7, TotalTokens: 14200, SessionID: "thread-1234567890", Event: "agent message streaming: waiting on rate-limit backoff window"}}
		base.Retrying = []observability.RetryRow{
			{ID: "MT-450", Attempt: 4, DueIn: 1250 * time.Millisecond, Error: "rate limit exhausted"},
			{ID: "MT-451", Attempt: 2, DueIn: 3900 * time.Millisecond, Error: "retrying after API timeout with jitter"},
			{ID: "MT-452", Attempt: 6, DueIn: 8100 * time.Millisecond, Error: "worker crashed\nrestarting cleanly"},
			{ID: "MT-453", Attempt: 1, DueIn: 11 * time.Second, Error: "fourth queued retry should also render after removing the top-three limit"},
		}
		return base
	case "credits_unlimited":
		base.Header.AgentCount = 1
		base.Header.ThroughputTPS = 42
		base.Header.RuntimeSeconds = 75
		base.Header.Tokens = domain.CodexTotals{InputTokens: 90, OutputTokens: 12, TotalTokens: 102}
		base.Header.RateLimits = &domain.RateLimits{
			LimitID:   "priority-tier",
			Primary:   &domain.RateLimitBucket{Remaining: ptrInt(100), Limit: ptrInt(100), ResetInSeconds: ptrInt(1)},
			Secondary: &domain.RateLimitBucket{Remaining: ptrInt(500), Limit: ptrInt(500), ResetInSeconds: ptrInt(1)},
			Credits:   &domain.RateLimitCredits{Unlimited: &unlimited},
		}
		base.Running = []observability.RunningRow{{ID: "MT-777", State: "running", PID: "4242", RuntimeSeconds: 75, TurnCount: 7, TotalTokens: 3200, SessionID: "thread-1234567890", Event: "thread token usage updated (in 90, out 12, total 102)"}}
		return base
	case "snapshot_unavailable":
		return observability.DashboardView{Mode: observability.DashboardModeUnavailable, UnavailableReason: "Snapshot unavailable"}
	case "orchestrator_snapshot_unavailable":
		return observability.DashboardView{Mode: observability.DashboardModeUnavailable, UnavailableReason: "Orchestrator snapshot unavailable"}
	case "offline":
		return observability.DashboardView{Mode: observability.DashboardModeOffline}
	default:
		panic(name)
	}
}

var noInt int

func ptrInt(v int) *int {
	return &v
}
