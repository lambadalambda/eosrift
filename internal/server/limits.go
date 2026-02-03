package server

import "sync"

type tokenTunnelLimiter struct {
	mu     sync.Mutex
	active map[int64]int
}

func newTokenTunnelLimiter() *tokenTunnelLimiter {
	return &tokenTunnelLimiter{
		active: make(map[int64]int),
	}
}

func (l *tokenTunnelLimiter) TryAcquire(tokenID int64, maxActive int) (release func(), ok bool) {
	if l == nil || maxActive <= 0 || tokenID <= 0 {
		return func() {}, true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active[tokenID] >= maxActive {
		return nil, false
	}

	l.active[tokenID]++
	released := false

	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()

		if released {
			return
		}
		released = true

		n := l.active[tokenID]
		if n <= 1 {
			delete(l.active, tokenID)
			return
		}
		l.active[tokenID] = n - 1
	}, true
}

