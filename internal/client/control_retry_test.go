package client

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func TestDialControlWithRetry_RetriesUntilSuccess(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	dial := func(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error) {
		n := attempts.Add(1)
		if n < 3 {
			return nil, nil, errors.New("dial failed")
		}
		return &websocket.Conn{}, &yamux.Session{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ws, session, err := dialControlWithRetryConfig(ctx, "ws://example/control", dial, dialRetryConfig{
		MinDelay: 1 * time.Millisecond,
		MaxDelay: 1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if ws == nil || session == nil {
		t.Fatalf("expected non-nil ws/session")
	}
	if got, want := attempts.Load(), int32(3); got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
}

func TestDialControlWithRetry_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	dial := func(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error) {
		attempts.Add(1)
		return nil, nil, errors.New("dial failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := dialControlWithRetryConfig(ctx, "ws://example/control", dial, dialRetryConfig{
		MinDelay: 1 * time.Millisecond,
		MaxDelay: 1 * time.Millisecond,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline exceeded", err)
	}
	if attempts.Load() < 1 {
		t.Fatalf("expected at least one attempt, got %d", attempts.Load())
	}
}
