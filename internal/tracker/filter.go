package tracker

import (
	"context"
	"strings"

	"github.com/Miss-you/go-symphony/internal/domain"
)

type filteredReader struct {
	reader TrackerReader
	match  string
}

// NewFilteredReader scopes a read-only tracker reader to one work item ID or identifier.
func NewFilteredReader(reader TrackerReader, match string) TrackerReader {
	return &filteredReader{reader: reader, match: normalizeMatch(match)}
}

func (r *filteredReader) ListCandidates(ctx context.Context) ([]domain.WorkItem, error) {
	items, err := r.reader.ListCandidates(ctx)
	if err != nil {
		return nil, err
	}
	return r.filter(items), nil
}

func (r *filteredReader) ListByStates(ctx context.Context, states []string) ([]domain.WorkItem, error) {
	items, err := r.reader.ListByStates(ctx, states)
	if err != nil {
		return nil, err
	}
	return r.filter(items), nil
}

func (r *filteredReader) RefreshByIDs(ctx context.Context, ids []string) ([]domain.WorkItem, error) {
	items, err := r.reader.RefreshByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return r.filter(items), nil
}

func (r *filteredReader) filter(items []domain.WorkItem) []domain.WorkItem {
	if r.match == "" {
		return []domain.WorkItem{}
	}
	filtered := make([]domain.WorkItem, 0, len(items))
	for _, item := range items {
		if normalizeMatch(item.ID) == r.match || normalizeMatch(item.Identifier) == r.match {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func normalizeMatch(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
