package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"eosrift.com/eosrift/internal/auth"
	"eosrift.com/eosrift/internal/control"
	"nhooyr.io/websocket"
)

func TestControl_RateLimitExceeded_RejectsSecondTunnelCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := auth.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	_, token, err := store.CreateToken(ctx, "owner")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain:              "tunnel.example.com",
		MaxTunnelCreatesPerMinute: 1,
	}, Dependencies{
		TokenValidator: store,
		TokenResolver:  store,
	}))
	t.Cleanup(srv.Close)

	ws1, session1 := dialTestControl(t, srv.URL)
	func() {
		defer func() {
			_ = session1.Close()
			_ = ws1.Close(websocket.StatusNormalClosure, "closed")
		}()

		stream, err := session1.OpenStream()
		if err != nil {
			t.Fatalf("open stream1: %v", err)
		}
		defer stream.Close()

		if err := control.WriteJSON(stream, control.CreateHTTPTunnelRequest{
			Type:      "http",
			Authtoken: token,
		}); err != nil {
			t.Fatalf("encode1: %v", err)
		}

		var resp control.CreateHTTPTunnelResponse
		if err := json.NewDecoder(stream).Decode(&resp); err != nil {
			t.Fatalf("decode1: %v", err)
		}
		if resp.Error != "" {
			t.Fatalf("resp1 error = %q, want empty", resp.Error)
		}
	}()

	ws2, session2 := dialTestControl(t, srv.URL)
	defer func() {
		_ = session2.Close()
		_ = ws2.Close(websocket.StatusNormalClosure, "closed")
	}()

	stream, err := session2.OpenStream()
	if err != nil {
		t.Fatalf("open stream2: %v", err)
	}
	defer stream.Close()

	if err := control.WriteJSON(stream, control.CreateHTTPTunnelRequest{
		Type:      "http",
		Authtoken: token,
	}); err != nil {
		t.Fatalf("encode2: %v", err)
	}

	var resp2 control.CreateHTTPTunnelResponse
	if err := json.NewDecoder(stream).Decode(&resp2); err != nil {
		t.Fatalf("decode2: %v", err)
	}

	if resp2.Error != "rate limit exceeded" {
		t.Fatalf("resp2 error = %q, want %q", resp2.Error, "rate limit exceeded")
	}
}
