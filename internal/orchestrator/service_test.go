package orchestrator

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/runner"
)

func TestServiceStartupPollShowsCheckingAndResetsAfterCycle(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	timers := newFakeTimerFactory(clock)
	svc := newService(testSettings(), serviceDeps{}, clock, timers, 20*time.Millisecond)
	t.Cleanup(svc.close)

	if len(timers.pending()) != 1 {
		t.Fatalf("pending timers after start = %d, want 1", len(timers.pending()))
	}

	timers.fireNext()
	snapshot := svc.snapshot()
	if !snapshot.Polling.Checking {
		t.Fatalf("checking = false, want true after initial tick")
	}
	if snapshot.Polling.NextPollAt != nil {
		t.Fatalf("next poll should be nil while checking")
	}

	timers.fireNext()
	snapshot = svc.snapshot()
	if snapshot.Polling.Checking {
		t.Fatalf("checking = true, want false after poll cycle")
	}
	if snapshot.Polling.NextPollAt == nil {
		t.Fatalf("next poll should be scheduled after poll cycle")
	}
}

func TestRequestRefreshCoalescesAndIgnoresStaleTick(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	timers := newFakeTimerFactory(clock)
	svc := newService(testSettings(), serviceDeps{}, clock, timers, 20*time.Millisecond)
	t.Cleanup(svc.close)

	initialTick := timers.peekNext()
	got := svc.requestRefresh()
	if got.Coalesced != true {
		t.Fatalf("initial refresh coalesced = %v, want true because immediate poll already queued", got.Coalesced)
	}

	timers.fireNext()
	if !svc.snapshot().Polling.Checking {
		t.Fatalf("service should be checking after first tick")
	}
	got = svc.requestRefresh()
	if got.Coalesced != true {
		t.Fatalf("refresh while checking coalesced = %v, want true", got.Coalesced)
	}

	timers.fireTimer(initialTick)
	if !svc.snapshot().Polling.Checking {
		t.Fatalf("stale tick should not clear checking state")
	}
}

func TestRunCompletedSchedulesRetryTimerAndRedispatches(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	timers := newFakeTimerFactory(clock)
	item := testItem("item", "MT-1", "In Progress", nil, clock.Now())

	var started []startRunRequest
	svc := newService(testSettings(), serviceDeps{
		refreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			if len(ids) != 1 || ids[0] != item.ID {
				t.Fatalf("refresh ids = %v, want [%s]", ids, item.ID)
			}
			return []domain.WorkItem{item}, nil
		},
		startRun: func(_ context.Context, req startRunRequest) (startRunResult, error) {
			started = append(started, req)
			return startRunResult{Handle: req.Item.ID, WorkerHost: req.PreferredHost}, nil
		},
		admitRun: func(_ string) (string, bool) { return "worker-a", true },
	}, clock, timers, 20*time.Millisecond)
	t.Cleanup(svc.close)

	// Move the startup poll out of the way so the retry timer becomes the next due timer.
	timers.fireNext()
	timers.fireNext()

	svc.state.running[item.ID] = runningEntry{
		Item:          item,
		StartedAt:     clock.Now(),
		WorkerHost:    "worker-a",
		Attempt:       0,
		Handle:        "handle-1",
		SessionID:     "session-1",
		WorkspacePath: "/tmp/mt1",
	}
	svc.state.claimed[item.ID] = claimedEntry{ItemID: item.ID, ItemIdentifier: item.Identifier, ClaimedAt: clock.Now()}

	svc.applyRunEvent(domain.RunEvent{
		Kind:           domain.RunEventRunCompleted,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             clock.Now(),
	})

	if got := svc.snapshot().Retrying; len(got) != 1 || got[0].Attempt != 1 {
		t.Fatalf("retry snapshot = %+v, want one continuation retry attempt", got)
	}

	timers.fireNext()

	if len(started) != 1 {
		t.Fatalf("startRun calls = %d, want 1 after retry timer fires", len(started))
	}
	if started[0].Attempt != 1 {
		t.Fatalf("retry attempt = %d, want 1", started[0].Attempt)
	}
	if _, ok := svc.state.running[item.ID]; !ok {
		t.Fatalf("item should be running again after retry dispatch")
	}
	if _, ok := svc.state.retrying[item.ID]; ok {
		t.Fatalf("retry entry should be cleared after redispatch")
	}
}

func TestDispatchOrderingAndGating(t *testing.T) {
	state := newSchedulerState(testSettings(), newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)))
	now := state.clock.Now()

	p1 := 1
	p2 := 2
	p3 := 3
	candidates := []domain.WorkItem{
		testItem("blocked", "MT-9", "Todo", &p1, now.Add(-3*time.Hour), blockedBy("Done")),
		testItem("todo-blocked", "MT-2", "Todo", &p1, now.Add(-4*time.Hour), blockedBy("In Progress")),
		testItem("routable-false", "MT-3", "In Progress", &p2, now.Add(-2*time.Hour), withRoutable(false)),
		testItem("later", "MT-4", "In Progress", &p2, now.Add(-1*time.Hour)),
		testItem("first", "MT-1", "In Progress", &p1, now.Add(-5*time.Hour)),
		testItem("claimed", "MT-5", "In Progress", &p1, now.Add(-30*time.Minute)),
		testItem("running", "MT-6", "In Progress", &p1, now.Add(-20*time.Minute)),
		testItem("backlog", "MT-7", "Backlog", &p3, now.Add(-10*time.Minute)),
	}

	state.claimed["claimed"] = claimedEntry{ItemID: "claimed", ItemIdentifier: "MT-5", ClaimedAt: now}
	state.running["running"] = runningEntry{Item: testItem("running", "MT-6", "In Progress", &p1, now.Add(-20*time.Minute)), StartedAt: now}

	var started []startRunRequest
	deps := serviceDeps{
		refreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			var refreshed []domain.WorkItem
			for _, item := range candidates {
				if slices.Contains(ids, item.ID) {
					refreshed = append(refreshed, item)
				}
			}
			return refreshed, nil
		},
		startRun: func(_ context.Context, req startRunRequest) (startRunResult, error) {
			started = append(started, req)
			return startRunResult{Handle: req.Item.ID, WorkerHost: req.PreferredHost}, nil
		},
		admitRun: func(_ string) (string, bool) { return "", true },
	}

	state.processCandidates(context.Background(), deps, candidates)

	if got := slices.Collect(slices.Values([]string{started[0].Item.ID, started[1].Item.ID})); !slices.Equal(got, []string{"first", "blocked"}) {
		t.Fatalf("started order = %v, want [first blocked]", got)
	}
}

func TestDispatchAdmitsHostOncePerDispatch(t *testing.T) {
	state := newSchedulerState(testSettings(), newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)))
	item := testItem("first", "MT-1", "In Progress", nil, state.clock.Now())

	admitCalls := 0
	deps := serviceDeps{
		refreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			return []domain.WorkItem{item}, nil
		},
		startRun: func(_ context.Context, req startRunRequest) (startRunResult, error) {
			return startRunResult{Handle: req.Item.ID, WorkerHost: req.PreferredHost}, nil
		},
		admitRun: func(_ string) (string, bool) {
			admitCalls++
			return "worker-a", true
		},
	}

	state.processCandidates(context.Background(), deps, []domain.WorkItem{item})

	if admitCalls != 1 {
		t.Fatalf("admitRun calls = %d, want 1 per dispatch", admitCalls)
	}
}

func TestHostLoadsDerivedFromRunningEntries(t *testing.T) {
	state := newSchedulerState(testSettings(), newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)))
	state.running["a1"] = runningEntry{Item: testItem("a1", "MT-1", "In Progress", nil, state.clock.Now()), WorkerHost: "worker-a"}
	state.running["a2"] = runningEntry{Item: testItem("a2", "MT-2", "In Progress", nil, state.clock.Now()), WorkerHost: "worker-a"}
	state.running["b1"] = runningEntry{Item: testItem("b1", "MT-3", "In Progress", nil, state.clock.Now()), WorkerHost: "worker-b"}
	state.running["local"] = runningEntry{Item: testItem("local", "MT-4", "In Progress", nil, state.clock.Now()), WorkerHost: ""}

	got := state.hostLoads()
	want := []runner.HostLoad{{Host: "worker-a", Running: 2}, {Host: "worker-b", Running: 1}}
	if !slices.Equal(got, want) {
		t.Fatalf("host loads = %+v, want %+v", got, want)
	}
}

func TestDispatchUsesRunnerHostSelection(t *testing.T) {
	state := newSchedulerState(testSettings(), newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)))
	state.running["busy"] = runningEntry{Item: testItem("busy", "MT-BUSY", "In Progress", nil, state.clock.Now()), WorkerHost: "worker-a"}
	item := testItem("first", "MT-1", "In Progress", nil, state.clock.Now())
	capOne := 1

	var started []startRunRequest
	deps := serviceDeps{
		hostSelection: &runner.HostSelection{Hosts: []string{"worker-a", "worker-b"}, MaxPerHost: &capOne},
		startRun: func(_ context.Context, req startRunRequest) (startRunResult, error) {
			started = append(started, req)
			return startRunResult{Handle: req.Item.ID, WorkerHost: req.PreferredHost}, nil
		},
	}

	if !state.dispatchItem(context.Background(), deps, item, 0, "") {
		t.Fatal("dispatchItem returned false, want worker-b admission")
	}
	if len(started) != 1 {
		t.Fatalf("start calls = %d, want 1", len(started))
	}
	if started[0].PreferredHost != "worker-b" {
		t.Fatalf("start preferred host = %q, want worker-b", started[0].PreferredHost)
	}
	if state.running[item.ID].WorkerHost != "worker-b" {
		t.Fatalf("running host = %q, want worker-b", state.running[item.ID].WorkerHost)
	}
}

func TestNewServiceDefaultsToRunnerHostSelectionFromSettings(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	timers := newFakeTimerFactory(clock)
	settings := testSettings()
	capOne := 1
	settings.Worker = config.WorkerSettings{
		SSHHosts:                   []string{"worker-a", "worker-b"},
		MaxConcurrentAgentsPerHost: &capOne,
	}

	var started []startRunRequest
	svc := newService(settings, serviceDeps{
		startRun: func(_ context.Context, req startRunRequest) (startRunResult, error) {
			started = append(started, req)
			return startRunResult{Handle: req.Item.ID, WorkerHost: req.PreferredHost}, nil
		},
	}, clock, timers, 20*time.Millisecond)
	t.Cleanup(svc.close)

	svc.state.running["busy"] = runningEntry{
		Item:       testItem("busy", "MT-BUSY", "In Progress", nil, clock.Now()),
		StartedAt:  clock.Now(),
		WorkerHost: "worker-a",
	}
	item := testItem("first", "MT-1", "In Progress", nil, clock.Now())

	if !svc.state.dispatchItem(context.Background(), svc.deps, item, 0, "") {
		t.Fatal("dispatchItem returned false, want worker-b admission from service settings")
	}
	if len(started) != 1 || started[0].PreferredHost != "worker-b" {
		t.Fatalf("started = %+v, want runner host selection to choose worker-b", started)
	}
}

func TestRunCompletedAndFailedScheduleRetryLineage(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	state := newSchedulerState(testSettings(), clock)

	state.running["done"] = runningEntry{
		Item:          testItem("done", "MT-1", "In Progress", nil, clock.Now()),
		Attempt:       0,
		StartedAt:     clock.Now(),
		WorkerHost:    "worker-a",
		WorkspacePath: "/tmp/mt1",
	}
	state.claimed["done"] = claimedEntry{ItemID: "done", ItemIdentifier: "MT-1", ClaimedAt: clock.Now()}
	state.applyRunEvent(domain.RunEvent{Kind: domain.RunEventRunCompleted, ItemID: "done", ItemIdentifier: "MT-1", At: clock.Now()})

	retry := state.retrying["done"]
	if retry.Attempt != 1 {
		t.Fatalf("continuation attempt = %d, want 1", retry.Attempt)
	}
	if retry.DueAt.Sub(clock.Now()) != time.Second {
		t.Fatalf("continuation delay = %v, want 1s", retry.DueAt.Sub(clock.Now()))
	}
	if _, ok := state.claimed["done"]; !ok {
		t.Fatalf("claim should be retained for continuation retry")
	}

	state.running["fail"] = runningEntry{
		Item:      testItem("fail", "MT-2", "In Progress", nil, clock.Now()),
		Attempt:   2,
		StartedAt: clock.Now(),
	}
	state.claimed["fail"] = claimedEntry{ItemID: "fail", ItemIdentifier: "MT-2", ClaimedAt: clock.Now()}
	state.applyRunEvent(domain.RunEvent{Kind: domain.RunEventRunFailed, ItemID: "fail", ItemIdentifier: "MT-2", At: clock.Now(), Err: errors.New("boom")})

	retry = state.retrying["fail"]
	if retry.Attempt != 3 {
		t.Fatalf("failure attempt = %d, want 3", retry.Attempt)
	}
	if retry.DueAt.Sub(clock.Now()) != 40*time.Second {
		t.Fatalf("failure delay = %v, want 40s", retry.DueAt.Sub(clock.Now()))
	}
}

func TestContinuationBlockedByCapacityFallsIntoFailureBackoff(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	state := newSchedulerState(testSettings(), clock)
	state.claimed["item"] = claimedEntry{ItemID: "item", ItemIdentifier: "MT-1", ClaimedAt: clock.Now()}
	state.retrying["item"] = retryEntry{
		ItemID:         "item",
		ItemIdentifier: "MT-1",
		Attempt:        1,
		DueAt:          clock.Now(),
		Nonce:          1,
	}

	deps := serviceDeps{
		refreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			return []domain.WorkItem{testItem("item", "MT-1", "In Progress", nil, clock.Now())}, nil
		},
		admitRun: func(_ string) (string, bool) { return "", false },
	}

	state.handleDueRetry(context.Background(), deps, "item", 1)
	retry := state.retrying["item"]
	if retry.Attempt != 2 {
		t.Fatalf("capacity-blocked continuation attempt = %d, want 2", retry.Attempt)
	}
	if retry.DueAt.Sub(clock.Now()) != 20*time.Second {
		t.Fatalf("capacity-blocked continuation delay = %v, want 20s", retry.DueAt.Sub(clock.Now()))
	}
}

func TestStaleRetryDeliveryIgnored(t *testing.T) {
	state := newSchedulerState(testSettings(), newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)))
	state.retrying["item"] = retryEntry{ItemID: "item", ItemIdentifier: "MT-1", Attempt: 2, DueAt: state.clock.Now(), Nonce: 9}
	state.handleDueRetry(context.Background(), serviceDeps{}, "item", 1)
	if state.retrying["item"].Nonce != 9 {
		t.Fatalf("stale retry should not replace newer retry entry")
	}
}

func TestReconcileAndStallRecovery(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	state := newSchedulerState(testSettings(), clock)
	p1 := 1
	state.running["active"] = runningEntry{Item: testItem("active", "MT-1", "In Progress", &p1, clock.Now()), StartedAt: clock.Now()}
	state.running["terminal"] = runningEntry{Item: testItem("terminal", "MT-2", "In Progress", &p1, clock.Now()), StartedAt: clock.Now()}
	state.running["stalled"] = runningEntry{Item: testItem("stalled", "MT-3", "In Progress", &p1, clock.Now()), Attempt: 0, StartedAt: clock.Now().Add(-10 * time.Minute)}
	state.claimed["active"] = claimedEntry{ItemID: "active", ItemIdentifier: "MT-1", ClaimedAt: clock.Now()}
	state.claimed["terminal"] = claimedEntry{ItemID: "terminal", ItemIdentifier: "MT-2", ClaimedAt: clock.Now()}
	state.claimed["stalled"] = claimedEntry{ItemID: "stalled", ItemIdentifier: "MT-3", ClaimedAt: clock.Now()}

	var stopped []stopRunRequest
	deps := serviceDeps{
		refreshItems: func(_ context.Context, ids []string) ([]domain.WorkItem, error) {
			return []domain.WorkItem{
				testItem("active", "MT-1", "In Progress", &p1, clock.Now()),
				testItem("terminal", "MT-2", "Done", &p1, clock.Now()),
			}, nil
		},
		stopRun: func(_ context.Context, req stopRunRequest) error {
			stopped = append(stopped, req)
			return nil
		},
	}

	state.reconcileStalled(context.Background(), deps)
	state.reconcileRunning(context.Background(), deps)

	if state.running["active"].Item.State != "In Progress" {
		t.Fatalf("active item state not refreshed")
	}
	if _, ok := state.claimed["terminal"]; ok {
		t.Fatalf("terminal claim should be released")
	}
	if state.retrying["stalled"].Attempt != 1 {
		t.Fatalf("stalled run should schedule failure retry attempt 1")
	}
	if len(stopped) != 2 {
		t.Fatalf("stop calls = %d, want 2", len(stopped))
	}
}

func TestSnapshotAndAggregateTotals(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	state := newSchedulerState(testSettings(), clock)

	state.running["b"] = runningEntry{
		Item:               testItem("b", "MT-2", "In Progress", nil, clock.Now()),
		StartedAt:          clock.Now().Add(-2 * time.Minute),
		LastEventKind:      domain.RunEventCodexEventReceived,
		LastEventMessage:   "b",
		CodexTotals:        domain.CodexTotals{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
		LastReportedTotals: domain.CodexTotals{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
	}
	state.running["a"] = runningEntry{
		Item:               testItem("a", "MT-1", "In Progress", nil, clock.Now()),
		StartedAt:          clock.Now().Add(-1 * time.Minute),
		CodexTotals:        domain.CodexTotals{InputTokens: 2, OutputTokens: 1, TotalTokens: 3},
		LastReportedTotals: domain.CodexTotals{InputTokens: 2, OutputTokens: 1, TotalTokens: 3},
	}
	state.retrying["b"] = retryEntry{ItemID: "b", ItemIdentifier: "MT-2", DueAt: clock.Now().Add(2 * time.Minute)}
	state.retrying["a"] = retryEntry{ItemID: "a", ItemIdentifier: "MT-1", DueAt: clock.Now().Add(1 * time.Minute)}

	state.applyRunEvent(domain.RunEvent{
		Kind:           domain.RunEventCodexEventReceived,
		ItemID:         "a",
		ItemIdentifier: "MT-1",
		At:             clock.Now(),
		CodexTotals:    domain.CodexTotals{InputTokens: 4, OutputTokens: 2, TotalTokens: 6},
		RateLimits:     &domain.RateLimits{LimitID: "codex"},
	})

	snapshot := state.snapshot()
	if got, want := snapshot.CodexTotals.TotalTokens, 3; got != want {
		t.Fatalf("aggregate total tokens = %d, want %d", got, want)
	}
	if snapshot.RateLimits == nil || snapshot.RateLimits.LimitID != "codex" {
		t.Fatalf("rate limits not updated")
	}
	if got, want := []string{snapshot.Running[0].ItemIdentifier, snapshot.Running[1].ItemIdentifier}, []string{"MT-1", "MT-2"}; !slices.Equal(got, want) {
		t.Fatalf("running order = %v, want %v", got, want)
	}
	if got, want := []string{snapshot.Retrying[0].ItemIdentifier, snapshot.Retrying[1].ItemIdentifier}, []string{"MT-1", "MT-2"}; !slices.Equal(got, want) {
		t.Fatalf("retry order = %v, want %v", got, want)
	}
}

func TestRetryScheduledEventIsMetadataOnly(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	state := newSchedulerState(testSettings(), clock)
	state.running["item"] = runningEntry{Item: testItem("item", "MT-1", "In Progress", nil, clock.Now()), StartedAt: clock.Now()}
	state.retrying["item"] = retryEntry{ItemID: "item", ItemIdentifier: "MT-1", Attempt: 1, DueAt: clock.Now(), Nonce: 5}

	state.applyRunEvent(domain.RunEvent{Kind: domain.RunEventRetryScheduled, ItemID: "item", ItemIdentifier: "MT-1", At: clock.Now(), Message: "scheduled"})

	if state.running["item"].LastEventKind != domain.RunEventRetryScheduled {
		t.Fatalf("retry_scheduled should stamp metadata")
	}
	if state.retrying["item"].Nonce != 5 {
		t.Fatalf("retry_scheduled should not rewrite retry entry")
	}
}

func testSettings() config.Settings {
	return config.Settings{
		Provider: config.ProviderSettings{
			ActiveStates:   []string{"Todo", "In Progress"},
			TerminalStates: []string{"Done", "Canceled"},
		},
		Polling: config.PollingSettings{IntervalMS: 30_000},
		Agent: config.AgentSettings{
			MaxConcurrentAgents:        3,
			MaxRetryBackoffMS:          300_000,
			MaxConcurrentAgentsByState: map[string]int{"in progress": 2, "todo": 1},
		},
		Codex: config.CodexSettings{StallTimeoutMS: 300_000},
	}
}

func testItem(id, identifier, state string, priority *int, createdAt time.Time, options ...itemOption) domain.WorkItem {
	item := domain.WorkItem{
		ID:         id,
		Identifier: identifier,
		Title:      identifier,
		State:      state,
		Priority:   priority,
		CreatedAt:  &createdAt,
	}
	for _, option := range options {
		option(&item)
	}
	return item
}

type itemOption func(*domain.WorkItem)

func withRoutable(v bool) itemOption {
	return func(item *domain.WorkItem) { item.Routable = &v }
}

func blockedBy(state string) itemOption {
	return func(item *domain.WorkItem) {
		item.BlockedBy = []domain.Blocker{{ID: "blocker", Identifier: "MT-B", State: state}}
	}
}

type fakeClock struct {
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now}
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type fakeTimerFactory struct {
	clock  *fakeClock
	timers []*fakeTimer
}

func newFakeTimerFactory(clock *fakeClock) *fakeTimerFactory {
	return &fakeTimerFactory{clock: clock}
}

func (f *fakeTimerFactory) AfterFunc(delay time.Duration, fn func()) timerHandle {
	timer := &fakeTimer{fireAt: f.clock.Now().Add(delay), fn: fn}
	f.timers = append(f.timers, timer)
	return timer
}

func (f *fakeTimerFactory) pending() []*fakeTimer {
	var pending []*fakeTimer
	for _, timer := range f.timers {
		if !timer.stopped && !timer.fired {
			pending = append(pending, timer)
		}
	}
	return pending
}

func (f *fakeTimerFactory) peekNext() *fakeTimer {
	pending := f.pending()
	next := pending[0]
	for _, timer := range pending[1:] {
		if timer.fireAt.Before(next.fireAt) {
			next = timer
		}
	}
	return next
}

func (f *fakeTimerFactory) fireNext() {
	f.fireTimer(f.peekNext())
}

func (f *fakeTimerFactory) fireTimer(timer *fakeTimer) {
	timer.fired = true
	f.clock.now = timer.fireAt
	timer.fn()
}

type fakeTimer struct {
	fireAt  time.Time
	fn      func()
	stopped bool
	fired   bool
}

func (t *fakeTimer) Stop() bool {
	wasActive := !t.stopped && !t.fired
	t.stopped = true
	return wasActive
}
