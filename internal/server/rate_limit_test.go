package server

import (
	"testing"
	"time"
)

func TestTokenRateLimiter_AllowAndRefill(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	l := newTokenRateLimiter(func() time.Time { return now })

	// 2 per minute: allow 2 immediate, deny 3rd.
	if !l.Allow(1, 2) {
		t.Fatalf("allow1=false, want true")
	}
	if !l.Allow(1, 2) {
		t.Fatalf("allow2=false, want true")
	}
	if l.Allow(1, 2) {
		t.Fatalf("allow3=true, want false")
	}

	// Refill rate is 1 token per 30 seconds.
	now = now.Add(29 * time.Second)
	if l.Allow(1, 2) {
		t.Fatalf("allow4=true before refill, want false")
	}

	now = now.Add(1 * time.Second)
	if !l.Allow(1, 2) {
		t.Fatalf("allow5=false after 30s, want true")
	}
	if l.Allow(1, 2) {
		t.Fatalf("allow6=true immediately, want false")
	}
}

