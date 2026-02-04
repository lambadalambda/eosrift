package client

import (
	"context"
	"time"

	"eosrift.com/eosrift/internal/mux"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type dialRetryConfig struct {
	MinDelay time.Duration
	MaxDelay time.Duration
}

func dialControlWithRetry(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error) {
	return dialControlWithRetryConfig(ctx, controlURL, dialControl, dialRetryConfig{
		MinDelay: 250 * time.Millisecond,
		MaxDelay: 5 * time.Second,
	})
}

func dialControlWithRetryConfig(
	ctx context.Context,
	controlURL string,
	dial func(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error),
	cfg dialRetryConfig,
) (*websocket.Conn, *yamux.Session, error) {
	if cfg.MinDelay <= 0 {
		cfg.MinDelay = 250 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 5 * time.Second
	}

	delay := cfg.MinDelay

	for {
		ws, session, err := dial(ctx, controlURL)
		if err == nil {
			return ws, session, nil
		}

		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}

		delay *= 2
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}
}

func dialControl(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error) {
	ws, _, err := websocket.Dial(ctx, controlURL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return nil, nil, err
	}

	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)

	session, err := yamux.Client(netConn, mux.QuietYamuxConfig())
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "yamux error")
		return nil, nil, err
	}

	return ws, session, nil
}
