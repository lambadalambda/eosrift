package server

import "testing"

func TestTokenTunnelLimiter(t *testing.T) {
	t.Parallel()

	l := newTokenTunnelLimiter()

	release1, ok := l.TryAcquire(1, 1)
	if !ok {
		t.Fatalf("acquire1 ok=false, want true")
	}

	if _, ok := l.TryAcquire(1, 1); ok {
		t.Fatalf("acquire2 ok=true, want false")
	}

	release1()
	release1() // idempotent

	if release2, ok := l.TryAcquire(1, 1); !ok {
		t.Fatalf("acquire3 ok=false, want true")
	} else {
		release2()
	}
}

