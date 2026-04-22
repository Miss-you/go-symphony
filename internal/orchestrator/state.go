package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/runner"
)

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type timerHandle interface {
	Stop() bool
}

type timerFactory interface {
	AfterFunc(delay time.Duration, fn func()) timerHandle
}

type realTimerFactory struct{}

func (realTimerFactory) AfterFunc(delay time.Duration, fn func()) timerHandle {
	return time.AfterFunc(delay, fn)
}

type serviceDeps struct {
	listCandidates func(context.Context) ([]domain.WorkItem, error)
	refreshItems   func(context.Context, []string) ([]domain.WorkItem, error)
	startRun       func(context.Context, startRunRequest) (startRunResult, error)
	stopRun        func(context.Context, stopRunRequest) error
	hostSelection  *runner.HostSelection
	admitRun       func(preferredHost string) (string, bool)
}

type startRunRequest struct {
	Item          domain.WorkItem
	Attempt       int
	PreferredHost string
}

type startRunResult struct {
	Handle        any
	WorkerHost    string
	WorkspacePath string
	SessionID     string
}

type stopRunRequest struct {
	ItemID           string
	ItemIdentifier   string
	Handle           any
	CleanupWorkspace bool
}

type refreshResult struct {
	Queued    bool
	Coalesced bool
}

type claimedEntry struct {
	ItemID         string
	ItemIdentifier string
	ClaimedAt      time.Time
}

type runningEntry struct {
	Item               domain.WorkItem
	Attempt            int
	StartedAt          time.Time
	LastEventAt        *time.Time
	LastEventKind      domain.RunEventKind
	LastEventMessage   string
	SessionID          string
	TurnCount          int
	WorkerHost         string
	WorkspacePath      string
	CodexTotals        domain.CodexTotals
	LastReportedTotals domain.CodexTotals
	Handle             any
}

type retryEntry struct {
	ItemID         string
	ItemIdentifier string
	Attempt        int
	DueAt          time.Time
	LastError      string
	WorkerHost     string
	WorkspacePath  string
	Nonce          uint64
}

type schedulerState struct {
	clock           clock
	interval        time.Duration
	maxConcurrent   int
	maxRetryBackoff time.Duration
	stallTimeout    time.Duration
	activeStates    map[string]struct{}
	terminalStates  map[string]struct{}
	stateLimits     map[string]int

	nextPollAt *time.Time
	checking   bool
	pollNonce  uint64
	retryNonce uint64

	claimed  map[string]claimedEntry
	running  map[string]runningEntry
	retrying map[string]retryEntry

	codexTotals domain.CodexTotals
	rateLimits  *domain.RateLimits
}

func newSchedulerState(settings config.Settings, clk clock) *schedulerState {
	if clk == nil {
		clk = realClock{}
	}

	return &schedulerState{
		clock:           clk,
		interval:        time.Duration(settings.Polling.IntervalMS) * time.Millisecond,
		maxConcurrent:   settings.Agent.MaxConcurrentAgents,
		maxRetryBackoff: time.Duration(settings.Agent.MaxRetryBackoffMS) * time.Millisecond,
		stallTimeout:    time.Duration(settings.Codex.StallTimeoutMS) * time.Millisecond,
		activeStates:    normalizeStateSet(settings.Provider.ActiveStates),
		terminalStates:  normalizeStateSet(settings.Provider.TerminalStates),
		stateLimits:     normalizeStateLimits(settings.Agent.MaxConcurrentAgentsByState),
		claimed:         make(map[string]claimedEntry),
		running:         make(map[string]runningEntry),
		retrying:        make(map[string]retryEntry),
	}
}

func (s *schedulerState) schedulePollAt(at time.Time) uint64 {
	s.pollNonce++
	at = at.UTC()
	s.nextPollAt = &at
	return s.pollNonce
}

func (s *schedulerState) beginPoll(nonce uint64) bool {
	if nonce != s.pollNonce {
		return false
	}
	s.checking = true
	s.nextPollAt = nil
	return true
}

func (s *schedulerState) finishPoll() {
	next := s.clock.Now().Add(s.interval).UTC()
	s.nextPollAt = &next
	s.checking = false
	s.pollNonce++
}

func (s *schedulerState) requestRefresh() refreshResult {
	now := s.clock.Now()
	if s.checking {
		return refreshResult{Queued: true, Coalesced: true}
	}
	if s.nextPollAt != nil && !s.nextPollAt.After(now) {
		return refreshResult{Queued: true, Coalesced: true}
	}
	s.schedulePollAt(now)
	return refreshResult{Queued: true, Coalesced: false}
}

func (s *schedulerState) processCandidates(ctx context.Context, deps serviceDeps, candidates []domain.WorkItem) {
	slog.Debug("processing candidates", "count", len(candidates), "running", len(s.running), "retrying", len(s.retrying))
	sorted := sortCandidates(candidates)
	dispatched := 0
	for _, item := range sorted {
		if !s.shouldDispatch(item) {
			slog.Debug("candidate skipped", "item_id", item.ID, "identifier", item.Identifier, "reason", "shouldDispatch=false")
			continue
		}
		refreshed, ok := s.revalidateCandidate(ctx, deps, item.ID)
		if !ok {
			slog.Debug("candidate revalidation failed", "item_id", item.ID, "identifier", item.Identifier)
			continue
		}
		if s.dispatchItem(ctx, deps, refreshed, 0, "") {
			dispatched++
			continue
		}
	}
	if dispatched > 0 {
		slog.Info("dispatched candidates", "count", dispatched)
	}
}

func (s *schedulerState) revalidateCandidate(ctx context.Context, deps serviceDeps, itemID string) (domain.WorkItem, bool) {
	if deps.refreshItems == nil {
		entry, ok := s.running[itemID]
		if ok {
			return entry.Item, true
		}
		return domain.WorkItem{}, false
	}
	items, err := deps.refreshItems(ctx, []string{itemID})
	if err != nil || len(items) == 0 {
		return domain.WorkItem{}, false
	}
	item := items[0]
	if !s.isRetryCandidate(item) {
		return domain.WorkItem{}, false
	}
	return item, true
}

func (s *schedulerState) dispatchItem(ctx context.Context, deps serviceDeps, item domain.WorkItem, attempt int, preferredHost string) bool {
	host, admitted := s.admitHost(deps, preferredHost)
	if !admitted {
		slog.Debug("dispatch rejected: no host admitted", "item_id", item.ID, "identifier", item.Identifier)
		return false
	}

	result := startRunResult{WorkerHost: host}
	if deps.startRun != nil {
		var err error
		result, err = deps.startRun(ctx, startRunRequest{Item: item, Attempt: attempt, PreferredHost: host})
		if err != nil {
			nextAttempt := attempt
			if nextAttempt <= 0 {
				nextAttempt = 1
			} else {
				nextAttempt++
			}
			slog.Error("dispatch failed: startRun error", "item_id", item.ID, "identifier", item.Identifier, "error", err, "attempt", nextAttempt)
			s.scheduleFailureRetry(item.ID, item.Identifier, nextAttempt, fmt.Sprintf("failed to start run: %v", err), host, "")
			return false
		}
	}

	s.claimed[item.ID] = claimedEntry{
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		ClaimedAt:      s.clock.Now(),
	}
	s.running[item.ID] = runningEntry{
		Item:          item,
		Attempt:       attempt,
		StartedAt:     s.clock.Now(),
		WorkerHost:    firstNonEmpty(result.WorkerHost, host),
		WorkspacePath: result.WorkspacePath,
		SessionID:     result.SessionID,
		Handle:        result.Handle,
	}
	delete(s.retrying, item.ID)
	slog.Info("run dispatched", "item_id", item.ID, "identifier", item.Identifier, "host", firstNonEmpty(result.WorkerHost, host), "attempt", attempt)
	return true
}

func (s *schedulerState) shouldDispatch(item domain.WorkItem) bool {
	if !s.isRetryCandidate(item) {
		return false
	}
	if _, ok := s.claimed[item.ID]; ok {
		return false
	}
	if _, ok := s.running[item.ID]; ok {
		return false
	}
	if s.maxConcurrent > 0 && len(s.running) >= s.maxConcurrent {
		return false
	}
	if limit := s.stateLimit(item.State); limit > 0 && s.runningCountForState(item.State) >= limit {
		return false
	}
	return true
}

func (s *schedulerState) isRetryCandidate(item domain.WorkItem) bool {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Identifier) == "" || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.State) == "" {
		return false
	}
	if s.isTerminal(item.State) || !s.isActive(item.State) {
		return false
	}
	if item.Routable != nil && !*item.Routable {
		return false
	}
	return !blockedTodo(item, s.terminalStates)
}

func (s *schedulerState) runningCountForState(state string) int {
	normalized := normalizeState(state)
	count := 0
	for _, entry := range s.running {
		if normalizeState(entry.Item.State) == normalized {
			count++
		}
	}
	return count
}

func (s *schedulerState) stateLimit(state string) int {
	return s.stateLimits[normalizeState(state)]
}

func (s *schedulerState) isActive(state string) bool {
	_, ok := s.activeStates[normalizeState(state)]
	return ok
}

func (s *schedulerState) isTerminal(state string) bool {
	_, ok := s.terminalStates[normalizeState(state)]
	return ok
}

func (s *schedulerState) applyRunEvent(event domain.RunEvent) {
	entry, ok := s.running[event.ItemID]
	if !ok {
		return
	}

	if !event.At.IsZero() {
		at := event.At.UTC()
		entry.LastEventAt = &at
	}
	entry.LastEventKind = event.Kind
	entry.LastEventMessage = event.Message
	if event.WorkerHost != "" {
		entry.WorkerHost = event.WorkerHost
	}
	if event.WorkspacePath != "" {
		entry.WorkspacePath = event.WorkspacePath
	}
	if event.SessionID != "" {
		entry.SessionID = event.SessionID
	}
	if event.Kind == domain.RunEventTurnCompleted {
		entry.TurnCount++
	}
	if hasTotals(event.CodexTotals) {
		s.applyAggregateDelta(&entry, event.CodexTotals)
	}
	if event.RateLimits != nil {
		copied := *event.RateLimits
		s.rateLimits = &copied
	}

	switch event.Kind {
	case domain.RunEventRunCompleted:
		s.running[event.ItemID] = entry
		delete(s.running, event.ItemID)
		s.scheduleContinuationRetry(entry.Item.ID, entry.Item.Identifier, entry.WorkerHost, entry.WorkspacePath)
		return
	case domain.RunEventRunFailed:
		s.running[event.ItemID] = entry
		delete(s.running, event.ItemID)
		nextAttempt := nextFailureAttempt(entry.Attempt)
		lastError := event.Message
		if event.Err != nil {
			lastError = event.Err.Error()
		}
		s.scheduleFailureRetry(entry.Item.ID, entry.Item.Identifier, nextAttempt, lastError, entry.WorkerHost, entry.WorkspacePath)
		return
	case domain.RunEventRetryScheduled:
		// Metadata only; workers do not own retry queue mutation.
	}

	s.running[event.ItemID] = entry
}

func (s *schedulerState) handleDueRetry(ctx context.Context, deps serviceDeps, itemID string, nonce uint64) {
	entry, ok := s.retrying[itemID]
	if !ok || entry.Nonce != nonce {
		return
	}
	delete(s.retrying, itemID)

	item, status := s.refreshRetryItem(ctx, deps, itemID)
	switch status {
	case retryMissing:
		delete(s.claimed, itemID)
		return
	case retryTerminal:
		delete(s.claimed, itemID)
		s.stopWithIntent(ctx, deps, itemID, entry.ItemIdentifier, nil, true)
		return
	case retryIneligible:
		delete(s.claimed, itemID)
		s.stopWithIntent(ctx, deps, itemID, entry.ItemIdentifier, nil, false)
		return
	case retryRefreshFailed:
		s.scheduleFailureRetry(itemID, entry.ItemIdentifier, entry.Attempt+1, "retry refresh failed", entry.WorkerHost, entry.WorkspacePath)
		return
	}

	if s.maxConcurrent > 0 && len(s.running) >= s.maxConcurrent {
		s.scheduleFailureRetry(itemID, item.Identifier, entry.Attempt+1, "no available orchestrator slots", entry.WorkerHost, entry.WorkspacePath)
		return
	}
	if limit := s.stateLimit(item.State); limit > 0 && s.runningCountForState(item.State) >= limit {
		s.scheduleFailureRetry(itemID, item.Identifier, entry.Attempt+1, "no available orchestrator slots", entry.WorkerHost, entry.WorkspacePath)
		return
	}

	if !s.dispatchItem(ctx, deps, item, entry.Attempt, entry.WorkerHost) {
		if _, ok := s.retrying[itemID]; !ok {
			s.scheduleFailureRetry(itemID, item.Identifier, entry.Attempt+1, "no available orchestrator slots", entry.WorkerHost, entry.WorkspacePath)
		}
	}
}

func (s *schedulerState) reconcileStalled(ctx context.Context, deps serviceDeps) {
	if s.stallTimeout <= 0 {
		return
	}
	now := s.clock.Now()
	for itemID, entry := range cloneRunning(s.running) {
		lastActivity := entry.StartedAt
		if entry.LastEventAt != nil {
			lastActivity = *entry.LastEventAt
		}
		if now.Sub(lastActivity) <= s.stallTimeout {
			continue
		}
		duration := now.Sub(lastActivity).Round(time.Millisecond)
		slog.Warn("run stalled", "item_id", itemID, "identifier", entry.Item.Identifier, "duration", duration, "attempt", entry.Attempt)
		delete(s.running, itemID)
		s.stopWithIntent(ctx, deps, itemID, entry.Item.Identifier, entry.Handle, false)
		s.scheduleFailureRetry(itemID, entry.Item.Identifier, nextFailureAttempt(entry.Attempt), fmt.Sprintf("stalled for %s without worker activity", duration), entry.WorkerHost, entry.WorkspacePath)
	}
}

func (s *schedulerState) reconcileRunning(ctx context.Context, deps serviceDeps) {
	if len(s.running) == 0 || deps.refreshItems == nil {
		return
	}

	ids := make([]string, 0, len(s.running))
	for itemID := range s.running {
		ids = append(ids, itemID)
	}
	refreshed, err := deps.refreshItems(ctx, ids)
	if err != nil {
		slog.Error("refresh running items failed", "error", err, "count", len(ids))
		return
	}
	visible := make(map[string]domain.WorkItem, len(refreshed))
	for _, item := range refreshed {
		visible[item.ID] = item
	}

	for itemID, entry := range cloneRunning(s.running) {
		item, ok := visible[itemID]
		if !ok {
			s.invalidateRun(ctx, deps, entry, false)
			continue
		}
		switch {
		case s.isTerminal(item.State):
			s.invalidateRun(ctx, deps, entry, true)
		case !s.isActive(item.State):
			s.invalidateRun(ctx, deps, entry, false)
		case item.Routable != nil && !*item.Routable:
			s.invalidateRun(ctx, deps, entry, false)
		default:
			entry.Item = item
			s.running[itemID] = entry
		}
	}
}

func (s *schedulerState) invalidateRun(ctx context.Context, deps serviceDeps, entry runningEntry, cleanup bool) {
	slog.Info("run invalidated", "item_id", entry.Item.ID, "identifier", entry.Item.Identifier, "state", entry.Item.State, "cleanup", cleanup)
	delete(s.running, entry.Item.ID)
	delete(s.claimed, entry.Item.ID)
	delete(s.retrying, entry.Item.ID)
	s.stopWithIntent(ctx, deps, entry.Item.ID, entry.Item.Identifier, entry.Handle, cleanup)
}

func (s *schedulerState) stopWithIntent(ctx context.Context, deps serviceDeps, itemID, identifier string, handle any, cleanup bool) {
	if deps.stopRun == nil {
		return
	}
	_ = deps.stopRun(ctx, stopRunRequest{
		ItemID:           itemID,
		ItemIdentifier:   identifier,
		Handle:           handle,
		CleanupWorkspace: cleanup,
	})
}

func (s *schedulerState) refreshRetryItem(ctx context.Context, deps serviceDeps, itemID string) (domain.WorkItem, retryStatus) {
	if deps.refreshItems == nil {
		return domain.WorkItem{}, retryRefreshFailed
	}
	items, err := deps.refreshItems(ctx, []string{itemID})
	if err != nil {
		return domain.WorkItem{}, retryRefreshFailed
	}
	if len(items) == 0 {
		return domain.WorkItem{}, retryMissing
	}
	item := items[0]
	if s.isTerminal(item.State) {
		return item, retryTerminal
	}
	if !s.isRetryCandidate(item) {
		return item, retryIneligible
	}
	return item, retryReady
}

func (s *schedulerState) scheduleContinuationRetry(itemID, identifier, workerHost, workspacePath string) {
	s.retryNonce++
	s.retrying[itemID] = retryEntry{
		ItemID:         itemID,
		ItemIdentifier: identifier,
		Attempt:        1,
		DueAt:          s.clock.Now().Add(time.Second),
		WorkerHost:     workerHost,
		WorkspacePath:  workspacePath,
		Nonce:          s.retryNonce,
	}
}

func (s *schedulerState) scheduleFailureRetry(itemID, identifier string, attempt int, lastError, workerHost, workspacePath string) {
	if attempt <= 0 {
		attempt = 1
	}
	s.retryNonce++
	delay := failureRetryDelay(attempt, s.maxRetryBackoff)
	s.retrying[itemID] = retryEntry{
		ItemID:         itemID,
		ItemIdentifier: identifier,
		Attempt:        attempt,
		DueAt:          s.clock.Now().Add(delay),
		LastError:      lastError,
		WorkerHost:     workerHost,
		WorkspacePath:  workspacePath,
		Nonce:          s.retryNonce,
	}
	slog.Warn("retry scheduled", "item_id", itemID, "identifier", identifier, "attempt", attempt, "delay", delay, "error", lastError)
}

func (s *schedulerState) applyAggregateDelta(entry *runningEntry, totals domain.CodexTotals) {
	delta := domain.CodexTotals{
		InputTokens:    max(0, totals.InputTokens-entry.LastReportedTotals.InputTokens),
		OutputTokens:   max(0, totals.OutputTokens-entry.LastReportedTotals.OutputTokens),
		TotalTokens:    max(0, totals.TotalTokens-entry.LastReportedTotals.TotalTokens),
		SecondsRunning: max(0, totals.SecondsRunning-entry.LastReportedTotals.SecondsRunning),
	}
	s.codexTotals.InputTokens += delta.InputTokens
	s.codexTotals.OutputTokens += delta.OutputTokens
	s.codexTotals.TotalTokens += delta.TotalTokens
	s.codexTotals.SecondsRunning += delta.SecondsRunning
	entry.CodexTotals = totals
	entry.LastReportedTotals = totals
}

func (s *schedulerState) snapshot() domain.Snapshot {
	running := make([]domain.ActiveRun, 0, len(s.running))
	for _, entry := range s.running {
		running = append(running, domain.ActiveRun{
			ItemID:           entry.Item.ID,
			ItemIdentifier:   entry.Item.Identifier,
			State:            entry.Item.State,
			WorkerHost:       entry.WorkerHost,
			WorkspacePath:    entry.WorkspacePath,
			SessionID:        entry.SessionID,
			TurnCount:        entry.TurnCount,
			StartedAt:        entry.StartedAt,
			LastEventAt:      entry.LastEventAt,
			LastEventKind:    entry.LastEventKind,
			LastEventMessage: entry.LastEventMessage,
			CodexTotals:      entry.CodexTotals,
		})
	}
	sort.Slice(running, func(i, j int) bool {
		if running[i].ItemIdentifier != running[j].ItemIdentifier {
			return running[i].ItemIdentifier < running[j].ItemIdentifier
		}
		if running[i].ItemID != running[j].ItemID {
			return running[i].ItemID < running[j].ItemID
		}
		return running[i].StartedAt.Before(running[j].StartedAt)
	})

	retrying := make([]domain.RetryEntry, 0, len(s.retrying))
	for _, entry := range s.retrying {
		retrying = append(retrying, domain.RetryEntry{
			ItemID:         entry.ItemID,
			ItemIdentifier: entry.ItemIdentifier,
			Attempt:        entry.Attempt,
			DueAt:          entry.DueAt,
			LastError:      entry.LastError,
			WorkerHost:     entry.WorkerHost,
			WorkspacePath:  entry.WorkspacePath,
		})
	}
	sort.Slice(retrying, func(i, j int) bool {
		if !retrying[i].DueAt.Equal(retrying[j].DueAt) {
			return retrying[i].DueAt.Before(retrying[j].DueAt)
		}
		if retrying[i].ItemIdentifier != retrying[j].ItemIdentifier {
			return retrying[i].ItemIdentifier < retrying[j].ItemIdentifier
		}
		return retrying[i].ItemID < retrying[j].ItemID
	})

	var rateLimits *domain.RateLimits
	if s.rateLimits != nil {
		copied := *s.rateLimits
		rateLimits = &copied
	}

	return domain.Snapshot{
		Running:  running,
		Retrying: retrying,
		Polling: domain.PollingState{
			Checking:   s.checking,
			NextPollAt: s.nextPollAt,
			Interval:   s.interval,
		},
		CodexTotals: s.codexTotals,
		RateLimits:  rateLimits,
	}
}

func (s *schedulerState) admitHost(deps serviceDeps, preferred string) (string, bool) {
	if deps.hostSelection != nil {
		return deps.hostSelection.Select(preferred, s.hostLoads())
	}
	if deps.admitRun == nil {
		return preferred, true
	}
	return deps.admitRun(preferred)
}

func (s *schedulerState) hostLoads() []runner.HostLoad {
	counts := make(map[string]int)
	for _, entry := range s.running {
		host := strings.TrimSpace(entry.WorkerHost)
		if host == "" {
			continue
		}
		counts[host]++
	}
	hosts := make([]string, 0, len(counts))
	for host := range counts {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	loads := make([]runner.HostLoad, 0, len(hosts))
	for _, host := range hosts {
		loads = append(loads, runner.HostLoad{Host: host, Running: counts[host]})
	}
	return loads
}

func sortCandidates(items []domain.WorkItem) []domain.WorkItem {
	sorted := append([]domain.WorkItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		pi, pj := priorityRank(sorted[i].Priority), priorityRank(sorted[j].Priority)
		if pi != pj {
			return pi < pj
		}
		ti, tj := createdAtKey(sorted[i]), createdAtKey(sorted[j])
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		idi := firstNonEmpty(sorted[i].Identifier, sorted[i].ID)
		idj := firstNonEmpty(sorted[j].Identifier, sorted[j].ID)
		return idi < idj
	})
	return sorted
}

func createdAtKey(item domain.WorkItem) time.Time {
	if item.CreatedAt != nil {
		return item.CreatedAt.UTC()
	}
	return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
}

func priorityRank(priority *int) int {
	if priority != nil && *priority >= 1 && *priority <= 4 {
		return *priority
	}
	return 5
}

func blockedTodo(item domain.WorkItem, terminalStates map[string]struct{}) bool {
	if normalizeState(item.State) != "todo" {
		return false
	}
	for _, blocker := range item.BlockedBy {
		if _, ok := terminalStates[normalizeState(blocker.State)]; !ok {
			return true
		}
	}
	return false
}

func normalizeStateSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeState(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func normalizeStateLimits(values map[string]int) map[string]int {
	out := make(map[string]int, len(values))
	for key, value := range values {
		if normalized := normalizeState(key); normalized != "" {
			out[normalized] = value
		}
	}
	return out
}

func normalizeState(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func failureRetryDelay(attempt int, maxBackoff time.Duration) time.Duration {
	delay := 10 * time.Second
	for i := 1; i < attempt; i++ {
		delay *= 2
		if maxBackoff > 0 && delay >= maxBackoff {
			return maxBackoff
		}
	}
	if maxBackoff > 0 && delay > maxBackoff {
		return maxBackoff
	}
	return delay
}

func nextFailureAttempt(current int) int {
	if current > 0 {
		return current + 1
	}
	return 1
}

func cloneRunning(in map[string]runningEntry) map[string]runningEntry {
	return mapsClone(in)
}

func mapsClone[M ~map[K]V, K comparable, V any](in M) M {
	out := make(M, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func hasTotals(totals domain.CodexTotals) bool {
	return totals != (domain.CodexTotals{})
}

type retryStatus int

const (
	retryReady retryStatus = iota
	retryMissing
	retryTerminal
	retryIneligible
	retryRefreshFailed
)
