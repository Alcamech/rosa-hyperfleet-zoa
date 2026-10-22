package api

import "testing"

func TestCoalesceReason_WhenReasonSet_ItShouldPreferReason(t *testing.T) {
	if got := coalesceReason("ROSAENG-1", "ROSAENG-2"); got != "ROSAENG-1" {
		t.Fatalf("got %q", got)
	}
}

func TestCoalesceReason_WhenOnlyLegacyJira_ItShouldUseJira(t *testing.T) {
	if got := coalesceReason("", "ROSAENG-99"); got != "ROSAENG-99" {
		t.Fatalf("got %q", got)
	}
}
