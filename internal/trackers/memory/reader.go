package memory

import (
	"context"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
)

// Reader is a deterministic in-memory tracker reader for local and test use.
type Reader struct {
	items []domain.WorkItem
}

// NewReader snapshots seeded work items for later read-only queries.
func NewReader(items []domain.WorkItem) *Reader {
	cloned := make([]domain.WorkItem, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneWorkItem(item))
	}
	return &Reader{items: cloned}
}

// ListCandidates returns the full seeded item set in seed order.
func (r *Reader) ListCandidates(context.Context) ([]domain.WorkItem, error) {
	return cloneWorkItems(r.items), nil
}

// ListByStates returns seeded items whose states match the requested normalized state names.
func (r *Reader) ListByStates(_ context.Context, states []string) ([]domain.WorkItem, error) {
	if len(states) == 0 {
		return []domain.WorkItem{}, nil
	}

	normalized := make(map[string]struct{}, len(states))
	for _, state := range states {
		if name := normalizeState(state); name != "" {
			normalized[name] = struct{}{}
		}
	}
	if len(normalized) == 0 {
		return []domain.WorkItem{}, nil
	}

	var matched []domain.WorkItem
	for _, item := range r.items {
		if _, ok := normalized[normalizeState(item.State)]; ok {
			matched = append(matched, cloneWorkItem(item))
		}
	}
	return matched, nil
}

// RefreshByIDs returns visible items in the same order as the requested IDs.
func (r *Reader) RefreshByIDs(_ context.Context, ids []string) ([]domain.WorkItem, error) {
	if len(ids) == 0 {
		return []domain.WorkItem{}, nil
	}

	index := make(map[string]domain.WorkItem, len(r.items))
	for _, item := range r.items {
		index[item.ID] = item
	}

	refreshed := make([]domain.WorkItem, 0, len(ids))
	for _, id := range ids {
		item, ok := index[id]
		if !ok {
			continue
		}
		refreshed = append(refreshed, cloneWorkItem(item))
	}
	return refreshed, nil
}

func cloneWorkItems(items []domain.WorkItem) []domain.WorkItem {
	cloned := make([]domain.WorkItem, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneWorkItem(item))
	}
	return cloned
}

func cloneWorkItem(item domain.WorkItem) domain.WorkItem {
	item.Labels = append([]string(nil), item.Labels...)
	item.BlockedBy = append([]domain.Blocker(nil), item.BlockedBy...)
	item.Priority = cloneInt(item.Priority)
	item.Routable = cloneBool(item.Routable)
	item.CreatedAt = cloneTime(item.CreatedAt)
	item.UpdatedAt = cloneTime(item.UpdatedAt)
	return item
}

func cloneInt(v *int) *int {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneBool(v *bool) *bool {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneTime(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}
