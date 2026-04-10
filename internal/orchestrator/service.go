package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
)

type service struct {
	mu sync.Mutex

	state *schedulerState
	deps  serviceDeps

	timers              timerFactory
	pollTransitionDelay time.Duration
	pollTimer           timerHandle
	pollCycleTimer      timerHandle
	retryTimers         map[string]scheduledRetry
	closed              bool
}

type scheduledRetry struct {
	nonce uint64
	timer timerHandle
}

func newService(settings config.Settings, deps serviceDeps, clk clock, timers timerFactory, transitionDelay time.Duration) *service {
	if timers == nil {
		timers = realTimerFactory{}
	}
	svc := &service{
		state:               newSchedulerState(settings, clk),
		deps:                deps,
		timers:              timers,
		pollTransitionDelay: transitionDelay,
		retryTimers:         make(map[string]scheduledRetry),
	}
	svc.schedulePoll(0)
	return svc
}

func (s *service) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.pollTimer != nil {
		s.pollTimer.Stop()
	}
	if s.pollCycleTimer != nil {
		s.pollCycleTimer.Stop()
	}
	for itemID, scheduled := range s.retryTimers {
		scheduled.timer.Stop()
		delete(s.retryTimers, itemID)
	}
}

func (s *service) snapshot() domain.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.snapshot()
}

func (s *service) requestRefresh() refreshResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.state.requestRefresh()
	if !result.Coalesced {
		s.schedulePoll(0)
	}
	return result
}

func (s *service) applyRunEvent(event domain.RunEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.applyRunEvent(event)
	s.syncRetryTimersLocked()
}

func (s *service) schedulePoll(delay time.Duration) {
	if s.closed {
		return
	}
	if s.pollTimer != nil {
		s.pollTimer.Stop()
	}
	nonce := s.state.schedulePollAt(s.state.clock.Now().Add(delay))
	s.pollTimer = s.timers.AfterFunc(delay, func() { s.handlePollTick(nonce) })
}

func (s *service) handlePollTick(nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.state.beginPoll(nonce) {
		return
	}
	if s.pollCycleTimer != nil {
		s.pollCycleTimer.Stop()
	}
	s.pollCycleTimer = s.timers.AfterFunc(s.pollTransitionDelay, func() { s.handlePollCycle(nonce) })
}

func (s *service) handlePollCycle(nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.state.checking || nonce != s.state.pollNonce {
		return
	}

	s.state.reconcileStalled(context.Background(), s.deps)
	s.state.reconcileRunning(context.Background(), s.deps)
	if s.deps.listCandidates != nil {
		if candidates, err := s.deps.listCandidates(context.Background()); err == nil {
			s.state.processCandidates(context.Background(), s.deps, candidates)
		}
	}
	s.syncRetryTimersLocked()
	s.state.finishPoll()
	s.schedulePoll(s.state.interval)
}

func (s *service) handleRetryTick(itemID string, nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if current, ok := s.retryTimers[itemID]; ok && current.nonce == nonce {
		delete(s.retryTimers, itemID)
	}
	s.state.handleDueRetry(context.Background(), s.deps, itemID, nonce)
	s.syncRetryTimersLocked()
}

func (s *service) syncRetryTimersLocked() {
	for itemID, scheduled := range s.retryTimers {
		entry, ok := s.state.retrying[itemID]
		if ok && scheduled.nonce == entry.Nonce {
			continue
		}
		scheduled.timer.Stop()
		delete(s.retryTimers, itemID)
	}

	now := s.state.clock.Now()
	for itemID, entry := range s.state.retrying {
		if current, ok := s.retryTimers[itemID]; ok && current.nonce == entry.Nonce {
			continue
		}
		delay := entry.DueAt.Sub(now)
		if delay < 0 {
			delay = 0
		}
		capturedItemID := itemID
		capturedNonce := entry.Nonce
		s.retryTimers[itemID] = scheduledRetry{
			nonce: entry.Nonce,
			timer: s.timers.AfterFunc(delay, func() { s.handleRetryTick(capturedItemID, capturedNonce) }),
		}
	}
}
