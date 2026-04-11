package dashboard

import (
	"testing"
	"time"
)

func TestRenderGateCoalescesAndFlushes(t *testing.T) {
	base := time.Date(2026, 4, 12, 1, 0, 0, 0, time.UTC)
	gate := NewRenderGate(time.Second)

	first := gate.Enqueue("first", "a", base)
	if !first.Render || first.Content != "first" {
		t.Fatalf("first decision = %#v, want immediate render", first)
	}
	duplicate := gate.Enqueue("first", "a", base.Add(100*time.Millisecond))
	if duplicate.Render || duplicate.Pending {
		t.Fatalf("duplicate decision = %#v, want suppressed", duplicate)
	}
	changed := gate.Enqueue("second", "b", base.Add(200*time.Millisecond))
	if changed.Render || !changed.Pending || changed.FlushAt == nil {
		t.Fatalf("changed decision = %#v, want pending", changed)
	}
	if content, ok := gate.Flush(base.Add(900 * time.Millisecond)); ok || content != "" {
		t.Fatalf("early flush = %q/%v, want none", content, ok)
	}
	if content, ok := gate.Flush(base.Add(time.Second)); !ok || content != "second" {
		t.Fatalf("interval flush = %q/%v, want second/true", content, ok)
	}
}

func TestRenderGateAllowsIdleRerender(t *testing.T) {
	base := time.Date(2026, 4, 12, 1, 0, 0, 0, time.UTC)
	gate := NewRenderGate(time.Second)
	_ = gate.Enqueue("frame", "fingerprint", base)
	if gate.ForceIdleRerender("fingerprint", base.Add(999*time.Millisecond)) {
		t.Fatalf("idle rerender before one second should be false")
	}
	if !gate.ForceIdleRerender("fingerprint", base.Add(time.Second)) {
		t.Fatalf("idle rerender after one second should be true")
	}
	decision := gate.Enqueue("frame", "fingerprint", base.Add(time.Second))
	if !decision.Render || decision.Content != "frame" {
		t.Fatalf("idle rerender enqueue = %#v, want duplicate frame rendered", decision)
	}
}
