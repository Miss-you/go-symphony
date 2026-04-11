package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
)

const defaultPollTransitionDelay = 20 * time.Millisecond

type Dependencies struct {
	ListCandidates func(context.Context) ([]domain.WorkItem, error)
	RefreshItems   func(context.Context, []string) ([]domain.WorkItem, error)
	StartRun       func(context.Context, StartRunRequest) (StartRunResult, error)
	StopRun        func(context.Context, StopRunRequest) error
}

type StartRunRequest struct {
	Item          domain.WorkItem
	Attempt       int
	PreferredHost string
}

type StartRunResult struct {
	Handle        any
	WorkerHost    string
	WorkspacePath string
	SessionID     string
}

type StopRunRequest struct {
	ItemID           string
	ItemIdentifier   string
	Handle           any
	CleanupWorkspace bool
}

type RefreshResult struct {
	Queued    bool
	Coalesced bool
}

type Service struct {
	mu     sync.Mutex
	inner  *service
	closed bool
}

func Start(settings config.Settings, deps Dependencies) *Service {
	innerDeps := serviceDeps{
		listCandidates: deps.ListCandidates,
		refreshItems:   deps.RefreshItems,
	}
	if deps.StartRun != nil {
		innerDeps.startRun = func(ctx context.Context, req startRunRequest) (startRunResult, error) {
			result, err := deps.StartRun(ctx, StartRunRequest(req))
			return startRunResult(result), err
		}
	}
	if deps.StopRun != nil {
		innerDeps.stopRun = func(ctx context.Context, req stopRunRequest) error {
			return deps.StopRun(ctx, StopRunRequest(req))
		}
	}
	return &Service{inner: newService(settings, innerDeps, realClock{}, realTimerFactory{}, defaultPollTransitionDelay)}
}

func (s *Service) Snapshot() domain.Snapshot {
	if s == nil || s.inner == nil {
		return domain.Snapshot{}
	}
	return s.inner.snapshot()
}

func (s *Service) RequestRefresh() RefreshResult {
	if s == nil || s.inner == nil {
		return RefreshResult{}
	}
	result := s.inner.requestRefresh()
	return RefreshResult(result)
}

func (s *Service) ApplyRunEvent(event domain.RunEvent) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.applyRunEvent(event)
}

func (s *Service) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.inner.close()
	return nil
}
