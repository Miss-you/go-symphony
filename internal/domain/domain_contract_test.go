package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestWorkItemContract(t *testing.T) {
	tpe := reflect.TypeOf(WorkItem{})

	assertFieldType(t, tpe, "ID", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Identifier", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Title", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Description", reflect.TypeOf(""))
	assertFieldType(t, tpe, "State", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Priority", reflect.PointerTo(reflect.TypeOf(int(0))))
	assertFieldType(t, tpe, "BranchName", reflect.TypeOf(""))
	assertFieldType(t, tpe, "URL", reflect.TypeOf(""))
	assertFieldType(t, tpe, "AssigneeID", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Labels", reflect.TypeOf([]string{}))
	assertFieldType(t, tpe, "BlockedBy", reflect.TypeOf([]Blocker{}))
	assertFieldType(t, tpe, "Routable", reflect.PointerTo(reflect.TypeOf(true)))
	assertFieldType(t, tpe, "CreatedAt", reflect.PointerTo(reflect.TypeOf(time.Time{})))
	assertFieldType(t, tpe, "UpdatedAt", reflect.PointerTo(reflect.TypeOf(time.Time{})))

	assertNoField(t, tpe, "Issue")
	assertNoField(t, tpe, "Linear")
	assertNoField(t, tpe, "Tracker")
	assertNoField(t, tpe, "ProjectSlug")
	assertNoField(t, tpe, "Metadata")
}

func TestRuntimeProjectionContracts(t *testing.T) {
	activeRun := reflect.TypeOf(ActiveRun{})
	assertFieldType(t, activeRun, "ItemID", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "ItemIdentifier", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "State", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "WorkerHost", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "WorkspacePath", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "SessionID", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "TurnCount", reflect.TypeOf(int(0)))
	assertFieldType(t, activeRun, "StartedAt", reflect.TypeOf(time.Time{}))
	assertFieldType(t, activeRun, "LastEventAt", reflect.PointerTo(reflect.TypeOf(time.Time{})))
	assertFieldType(t, activeRun, "LastEventKind", reflect.TypeOf(RunEventKind("")))
	assertFieldType(t, activeRun, "LastEventMessage", reflect.TypeOf(""))
	assertFieldType(t, activeRun, "CodexTotals", reflect.TypeOf(CodexTotals{}))
	assertNoField(t, activeRun, "PID")
	assertNoField(t, activeRun, "Ref")
	assertNoField(t, activeRun, "TimerRef")

	retryEntry := reflect.TypeOf(RetryEntry{})
	assertFieldType(t, retryEntry, "ItemID", reflect.TypeOf(""))
	assertFieldType(t, retryEntry, "ItemIdentifier", reflect.TypeOf(""))
	assertFieldType(t, retryEntry, "Attempt", reflect.TypeOf(int(0)))
	assertFieldType(t, retryEntry, "DueAt", reflect.TypeOf(time.Time{}))
	assertFieldType(t, retryEntry, "LastError", reflect.TypeOf(""))
	assertFieldType(t, retryEntry, "WorkerHost", reflect.TypeOf(""))
	assertFieldType(t, retryEntry, "WorkspacePath", reflect.TypeOf(""))

	pollingState := reflect.TypeOf(PollingState{})
	assertFieldType(t, pollingState, "Checking", reflect.TypeOf(true))
	assertFieldType(t, pollingState, "NextPollAt", reflect.PointerTo(reflect.TypeOf(time.Time{})))
	assertFieldType(t, pollingState, "Interval", reflect.TypeOf(time.Duration(0)))

	snapshot := reflect.TypeOf(Snapshot{})
	assertFieldType(t, snapshot, "Running", reflect.TypeOf([]ActiveRun{}))
	assertFieldType(t, snapshot, "Retrying", reflect.TypeOf([]RetryEntry{}))
	assertFieldType(t, snapshot, "Polling", reflect.TypeOf(PollingState{}))
	assertFieldType(t, snapshot, "CodexTotals", reflect.TypeOf(CodexTotals{}))
	assertFieldType(t, snapshot, "RateLimits", reflect.PointerTo(reflect.TypeOf(RateLimits{})))
	assertNoField(t, snapshot, "RunningMap")
	assertNoField(t, snapshot, "Claimed")
	assertNoField(t, snapshot, "TickTimerRef")
}

func TestRunEventContract(t *testing.T) {
	tpe := reflect.TypeOf(RunEvent{})

	assertFieldType(t, tpe, "Kind", reflect.TypeOf(RunEventKind("")))
	assertFieldType(t, tpe, "ItemID", reflect.TypeOf(""))
	assertFieldType(t, tpe, "ItemIdentifier", reflect.TypeOf(""))
	assertFieldType(t, tpe, "At", reflect.TypeOf(time.Time{}))
	assertFieldType(t, tpe, "Attempt", reflect.TypeOf(int(0)))
	assertFieldType(t, tpe, "WorkerHost", reflect.TypeOf(""))
	assertFieldType(t, tpe, "WorkspacePath", reflect.TypeOf(""))
	assertFieldType(t, tpe, "SessionID", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Message", reflect.TypeOf(""))
	assertFieldType(t, tpe, "Err", reflect.TypeOf((*error)(nil)).Elem())
	assertFieldType(t, tpe, "CodexTotals", reflect.TypeOf(CodexTotals{}))
	assertFieldType(t, tpe, "RateLimits", reflect.PointerTo(reflect.TypeOf(RateLimits{})))

	wantKinds := []RunEventKind{
		RunEventWorkspaceCreated,
		RunEventWorkspacePathDiscovered,
		RunEventRunnerHostSelected,
		RunEventCodexEventReceived,
		RunEventTurnCompleted,
		RunEventRunCompleted,
		RunEventRunFailed,
		RunEventRetryScheduled,
	}

	if len(wantKinds) != 8 {
		t.Fatalf("expected 8 stable run event kinds, got %d", len(wantKinds))
	}
}

func TestRateLimitContracts(t *testing.T) {
	totals := reflect.TypeOf(CodexTotals{})
	assertFieldType(t, totals, "InputTokens", reflect.TypeOf(int(0)))
	assertFieldType(t, totals, "OutputTokens", reflect.TypeOf(int(0)))
	assertFieldType(t, totals, "TotalTokens", reflect.TypeOf(int(0)))
	assertFieldType(t, totals, "SecondsRunning", reflect.TypeOf(int(0)))

	rateLimits := reflect.TypeOf(RateLimits{})
	assertFieldType(t, rateLimits, "LimitID", reflect.TypeOf(""))
	assertFieldType(t, rateLimits, "Primary", reflect.PointerTo(reflect.TypeOf(RateLimitBucket{})))
	assertFieldType(t, rateLimits, "Secondary", reflect.PointerTo(reflect.TypeOf(RateLimitBucket{})))
	assertFieldType(t, rateLimits, "Credits", reflect.PointerTo(reflect.TypeOf(RateLimitCredits{})))

	rateLimitBucket := reflect.TypeOf(RateLimitBucket{})
	assertFieldType(t, rateLimitBucket, "Remaining", reflect.PointerTo(reflect.TypeOf(int(0))))
	assertFieldType(t, rateLimitBucket, "Limit", reflect.PointerTo(reflect.TypeOf(int(0))))
	assertFieldType(t, rateLimitBucket, "ResetInSeconds", reflect.PointerTo(reflect.TypeOf(int(0))))

	rateLimitCredits := reflect.TypeOf(RateLimitCredits{})
	assertFieldType(t, rateLimitCredits, "HasCredits", reflect.PointerTo(reflect.TypeOf(true)))
	assertFieldType(t, rateLimitCredits, "Unlimited", reflect.PointerTo(reflect.TypeOf(true)))
	assertFieldType(t, rateLimitCredits, "Balance", reflect.PointerTo(reflect.TypeOf(float64(0))))
}

func assertFieldType(t *testing.T, tpe reflect.Type, fieldName string, want reflect.Type) {
	t.Helper()

	field, ok := tpe.FieldByName(fieldName)
	if !ok {
		t.Fatalf("expected field %q on %s", fieldName, tpe.Name())
	}
	if field.Type != want {
		t.Fatalf("field %s.%s: got %s want %s", tpe.Name(), fieldName, field.Type, want)
	}
}

func assertNoField(t *testing.T, tpe reflect.Type, fieldName string) {
	t.Helper()

	if _, ok := tpe.FieldByName(fieldName); ok {
		t.Fatalf("did not expect field %q on %s", fieldName, tpe.Name())
	}
}
