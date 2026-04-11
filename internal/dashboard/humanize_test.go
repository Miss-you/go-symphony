package dashboard

import "testing"

func TestFormatHelpers(t *testing.T) {
	if got := FormatCount(268500); got != "268,500" {
		t.Fatalf("count = %q", got)
	}
	if got := CompactSessionID("thread-abcdef1234567890"); got != "thre...567890" {
		t.Fatalf("session = %q", got)
	}
	if got := SanitizeInline("worker crashed\\nrestarting\ncleanly"); got != "worker crashed restarting cleanly" {
		t.Fatalf("sanitized = %q", got)
	}
	if got := Truncate("abcdefghijklmnopqrstuvwxyz", 10); got != "abcdefg..." {
		t.Fatalf("truncate = %q", got)
	}
}
