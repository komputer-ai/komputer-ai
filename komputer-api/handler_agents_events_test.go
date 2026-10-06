package main

import "testing"

// TestShouldBackfillEvents pins the rule that JSONL session backfill in
// getAgentEvents only runs for uncursored requests. A cursored poll
// (after) or seek (before/around) that comes back empty from Redis means
// "no events there" — backfilling would re-RPush the whole session into
// komputer-history:<name> (no dedupe) and return stale events, ignoring
// the cursor the caller asked for.
func TestShouldBackfillEvents(t *testing.T) {
	cases := []struct {
		name               string
		before             string
		after              string
		around             string
		wantShouldBackfill bool
	}{
		{"uncursored", "", "", "", true},
		{"before set", "2024-01-01T00:00:00Z", "", "", false},
		{"after set (polling)", "", "2024-01-01T00:00:00Z", "", false},
		{"around set", "", "", "2024-01-01T00:00:00Z", false},
		{"before and after set", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldBackfillEvents(tc.before, tc.after, tc.around)
			if got != tc.wantShouldBackfill {
				t.Errorf("shouldBackfillEvents(%q, %q, %q) = %v, want %v",
					tc.before, tc.after, tc.around, got, tc.wantShouldBackfill)
			}
		})
	}
}
