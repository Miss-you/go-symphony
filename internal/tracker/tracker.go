package tracker

import (
	"context"

	"github.com/Miss-you/go-symphony/internal/domain"
)

// TrackerReader is the provider-neutral, read-only tracker contract used by the core.
type TrackerReader interface {
	ListCandidates(context.Context) ([]domain.WorkItem, error)
	ListByStates(context.Context, []string) ([]domain.WorkItem, error)
	RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)
}
