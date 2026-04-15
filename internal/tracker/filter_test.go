package tracker

import (
	"context"
	"errors"
	"testing"

	"github.com/Miss-you/go-symphony/internal/domain"
)

func TestFilteredReaderMatchesCandidatesByIDOrIdentifier(t *testing.T) {
	t.Parallel()

	reader := NewFilteredReader(&staticReader{candidates: []domain.WorkItem{
		{ID: "item-1", Identifier: "ABC-1"},
		{ID: "item-2", Identifier: "ABC-2"},
	}}, "ABC-2")

	got, err := reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("ListCandidates error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "item-2" {
		t.Fatalf("ListCandidates = %#v, want only item-2", got)
	}

	reader = NewFilteredReader(&staticReader{candidates: []domain.WorkItem{
		{ID: "item-1", Identifier: "ABC-1"},
		{ID: "item-2", Identifier: "ABC-2"},
	}}, "item-1")

	got, err = reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("ListCandidates by ID error = %v", err)
	}
	if len(got) != 1 || got[0].Identifier != "ABC-1" {
		t.Fatalf("ListCandidates by ID = %#v, want only ABC-1", got)
	}
}

func TestFilteredReaderScopesStateAndRefreshReads(t *testing.T) {
	t.Parallel()

	reader := NewFilteredReader(&staticReader{
		byStates: []domain.WorkItem{
			{ID: "done-1", Identifier: "ABC-1", State: "Done"},
			{ID: "done-2", Identifier: "ABC-2", State: "Done"},
		},
		refresh: []domain.WorkItem{
			{ID: "item-1", Identifier: "ABC-1", State: "In Progress"},
			{ID: "item-2", Identifier: "ABC-2", State: "In Progress"},
		},
	}, "ABC-1")

	states, err := reader.ListByStates(context.Background(), []string{"Done"})
	if err != nil {
		t.Fatalf("ListByStates error = %v", err)
	}
	if len(states) != 1 || states[0].ID != "done-1" {
		t.Fatalf("ListByStates = %#v, want only done-1", states)
	}

	refreshed, err := reader.RefreshByIDs(context.Background(), []string{"item-1", "item-2"})
	if err != nil {
		t.Fatalf("RefreshByIDs error = %v", err)
	}
	if len(refreshed) != 1 || refreshed[0].ID != "item-1" {
		t.Fatalf("RefreshByIDs = %#v, want only item-1", refreshed)
	}
}

func TestFilteredReaderEmptyMatchAndErrorPropagation(t *testing.T) {
	t.Parallel()

	reader := NewFilteredReader(&staticReader{
		candidates: []domain.WorkItem{{ID: "item-1", Identifier: "ABC-1"}},
	}, "missing")

	got, err := reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("ListCandidates error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListCandidates len = %d, want 0", len(got))
	}

	wantErr := errors.New("boom")
	reader = NewFilteredReader(&staticReader{err: wantErr}, "ABC-1")
	if _, err := reader.ListCandidates(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("ListCandidates error = %v, want %v", err, wantErr)
	}
	if _, err := reader.ListByStates(context.Background(), []string{"Done"}); !errors.Is(err, wantErr) {
		t.Fatalf("ListByStates error = %v, want %v", err, wantErr)
	}
	if _, err := reader.RefreshByIDs(context.Background(), []string{"item-1"}); !errors.Is(err, wantErr) {
		t.Fatalf("RefreshByIDs error = %v, want %v", err, wantErr)
	}
}

type staticReader struct {
	candidates []domain.WorkItem
	byStates   []domain.WorkItem
	refresh    []domain.WorkItem
	err        error
}

func (r *staticReader) ListCandidates(context.Context) ([]domain.WorkItem, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]domain.WorkItem(nil), r.candidates...), nil
}

func (r *staticReader) ListByStates(context.Context, []string) ([]domain.WorkItem, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]domain.WorkItem(nil), r.byStates...), nil
}

func (r *staticReader) RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]domain.WorkItem(nil), r.refresh...), nil
}
