package server

import (
	"sync"
	"time"
)

type tokenRateLimiter struct {
	mu sync.Mutex

	now func() time.Time

	// buckets maps token id -> token bucket state.
	buckets map[int64]*tokenBucket
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func newTokenRateLimiter(now func() time.Time) *tokenRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &tokenRateLimiter{
		now:     now,
		buckets: make(map[int64]*tokenBucket),
	}
}

// Allow consumes one token if available and returns true.
//
// limitPerMinute controls both steady-state rate and burst capacity.
// - 0 or less means unlimited.
func (l *tokenRateLimiter) Allow(tokenID int64, limitPerMinute int) bool {
	if l == nil || limitPerMinute <= 0 || tokenID <= 0 {
		return true
	}

	// Token bucket: capacity = limitPerMinute, refill = limitPerMinute / 60 tokens/sec.
	capacity := float64(limitPerMinute)
	refillPerSecond := capacity / 60.0

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	b := l.buckets[tokenID]
	if b == nil {
		b = &tokenBucket{
			tokens: capacity,
			last:   now,
		}
		l.buckets[tokenID] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = minFloat64(capacity, b.tokens+elapsed*refillPerSecond)
		b.last = now
	} else if elapsed < 0 {
		// Clock moved backwards; keep state but update timestamp to avoid huge refills later.
		b.last = now
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens -= 1
	return true
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

