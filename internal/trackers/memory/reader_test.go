package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
)

func TestReaderImplementsTrackerReader(t *testing.T) {
	t.Parallel()

	var _ tracker.TrackerReader = (*Reader)(nil)
}

func TestListCandidatesReturnsSeededItemsAndDeepCopies(t *testing.T) {
	t.Parallel()

	reader := NewReader([]domain.WorkItem{testItem("item-1", "MT-1", "Todo")})

	got, err := reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("ListCandidates error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListCandidates len = %d, want 1", len(got))
	}

	got[0].Labels[0] = "mutated"
	got[0].BlockedBy[0].State = "Closed"
	*got[0].Priority = 99
	*got[0].Routable = false
	createdAt := got[0].CreatedAt.Add(6 * time.Hour)
	updatedAt := got[0].UpdatedAt.Add(6 * time.Hour)
	*got[0].CreatedAt = createdAt
	*got[0].UpdatedAt = updatedAt

	again, err := reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("second ListCandidates error = %v, want nil", err)
	}
	if again[0].Labels[0] != "backend" {
		t.Fatalf("Labels leaked mutation: got %q", again[0].Labels[0])
	}
	if again[0].BlockedBy[0].State != "In Progress" {
		t.Fatalf("BlockedBy leaked mutation: got %q", again[0].BlockedBy[0].State)
	}
	if priority := derefInt(t, again[0].Priority); priority != 2 {
		t.Fatalf("Priority leaked mutation: got %d, want 2", priority)
	}
	if routable := derefBool(t, again[0].Routable); !routable {
		t.Fatalf("Routable leaked mutation: got %v, want true", routable)
	}
	if got := derefTime(t, again[0].CreatedAt); !got.Equal(testCreatedAt()) {
		t.Fatalf("CreatedAt leaked mutation: got %v, want %v", got, testCreatedAt())
	}
	if got := derefTime(t, again[0].UpdatedAt); !got.Equal(testUpdatedAt()) {
		t.Fatalf("UpdatedAt leaked mutation: got %v, want %v", got, testUpdatedAt())
	}
}

func TestListByStatesNormalizesInputAndHandlesEmpty(t *testing.T) {
	t.Parallel()

	reader := NewReader([]domain.WorkItem{
		testItem("item-1", "MT-1", "Todo"),
		testItem("item-2", "MT-2", "In Progress"),
		testItem("item-3", "MT-3", "Closed"),
	})

	got, err := reader.ListByStates(context.Background(), []string{" todo ", "IN PROGRESS"})
	if err != nil {
		t.Fatalf("ListByStates error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListByStates len = %d, want 2", len(got))
	}
	if got[0].ID != "item-1" || got[1].ID != "item-2" {
		t.Fatalf("ListByStates IDs = [%s %s], want [item-1 item-2]", got[0].ID, got[1].ID)
	}

	got, err = reader.ListByStates(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListByStates empty error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListByStates empty len = %d, want 0", len(got))
	}
}

func TestRefreshByIDsPreservesRequestOrderAndHandlesEmpty(t *testing.T) {
	t.Parallel()

	reader := NewReader([]domain.WorkItem{
		testItem("item-a", "MT-A", "Todo"),
		testItem("item-b", "MT-B", "Todo"),
		testItem("item-c", "MT-C", "Todo"),
	})

	got, err := reader.RefreshByIDs(context.Background(), []string{"item-c", "missing", "item-a"})
	if err != nil {
		t.Fatalf("RefreshByIDs error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("RefreshByIDs len = %d, want 2", len(got))
	}
	if got[0].ID != "item-c" || got[1].ID != "item-a" {
		t.Fatalf("RefreshByIDs IDs = [%s %s], want [item-c item-a]", got[0].ID, got[1].ID)
	}

	got, err = reader.RefreshByIDs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("RefreshByIDs empty error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("RefreshByIDs empty len = %d, want 0", len(got))
	}
}

func testItem(id, identifier, state string) domain.WorkItem {
	return domain.WorkItem{
		ID:          id,
		Identifier:  identifier,
		Title:       "Fix the thing",
		Description: "Adapter contract test fixture",
		State:       state,
		Priority:    intPtr(2),
		BranchName:  "feature/" + identifier,
		URL:         "https://example.test/" + identifier,
		AssigneeID:  "worker-1",
		Labels:      []string{"backend", "urgent"},
		BlockedBy: []domain.Blocker{
			{ID: "blocker-1", Identifier: "BLK-1", State: "In Progress"},
		},
		Routable:  boolPtr(true),
		CreatedAt: timePtr(testCreatedAt()),
		UpdatedAt: timePtr(testUpdatedAt()),
	}
}

func testCreatedAt() time.Time {
	return time.Date(2026, time.April, 11, 1, 0, 0, 0, time.UTC)
}

func testUpdatedAt() time.Time {
	return time.Date(2026, time.April, 11, 2, 0, 0, 0, time.UTC)
}

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func timePtr(v time.Time) *time.Time { return &v }

func derefInt(t *testing.T, v *int) int {
	t.Helper()
	if v == nil {
		t.Fatal("got nil int pointer")
	}
	return *v
}

func derefBool(t *testing.T, v *bool) bool {
	t.Helper()
	if v == nil {
		t.Fatal("got nil bool pointer")
	}
	return *v
}

func derefTime(t *testing.T, v *time.Time) time.Time {
	t.Helper()
	if v == nil {
		t.Fatal("got nil time pointer")
	}
	return *v
}
