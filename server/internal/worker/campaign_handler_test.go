package worker

import (
	"testing"
	"time"
)

func TestShardRecipients(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		recipients []string
		shards     int
		wantShards int
	}{
		{"single shard", []string{"a", "b", "c"}, 1, 1},
		{"zero collapses to one", []string{"a", "b"}, 0, 1},
		{"even split", []string{"a", "b", "c", "d"}, 2, 2},
		{"more shards than recipients drops empties", []string{"a", "b"}, 5, 2},
		{"empty input", nil, 3, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shardRecipients(tc.recipients, tc.shards)
			if len(got) != tc.wantShards {
				t.Fatalf("shardRecipients(%v, %d) = %d shards, want %d",
					tc.recipients, tc.shards, len(got), tc.wantShards)
			}

			// Every recipient must appear exactly once across the shards:
			// dropping one silently loses an email, duplicating one sends twice.
			seen := map[string]int{}
			for _, shard := range got {
				for _, r := range shard {
					seen[r]++
				}
			}
			if len(seen) != len(tc.recipients) {
				t.Errorf("got %d distinct recipients, want %d", len(seen), len(tc.recipients))
			}
			for r, n := range seen {
				if n != 1 {
					t.Errorf("recipient %q appears %d times, want 1", r, n)
				}
			}
		})
	}
}

func TestCampaignConcurrencyNeverExceedsRecipients(t *testing.T) {
	t.Parallel()

	if got := campaignConcurrency(3); got != 3 {
		t.Errorf("campaignConcurrency(3) = %d, want 3", got)
	}
	if got := campaignConcurrency(0); got != 0 {
		t.Errorf("campaignConcurrency(0) = %d, want 0", got)
	}
	if got := campaignConcurrency(1000); got <= 0 || got > 1000 {
		t.Errorf("campaignConcurrency(1000) = %d, want a positive bound", got)
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{-5 * time.Second, "0s"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1.5m"},
		{2 * time.Hour, "2.0h"},
	} {
		if got := formatDuration(tc.d); got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
