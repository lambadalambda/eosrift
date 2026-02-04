package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"

	"eosrift.com/eosrift/internal/auth"
	"eosrift.com/eosrift/internal/control"
	"nhooyr.io/websocket"
)

func TestControlTCP_RequestedPort_AutoReservesAndEnforcesOwnership(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := auth.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	_, tokenA, err := store.CreateToken(ctx, "a")
	if err != nil {
		t.Fatalf("create token a: %v", err)
	}
	_, tokenB, err := store.CreateToken(ctx, "b")
	if err != nil {
		t.Fatalf("create token b: %v", err)
	}

	tmpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp: %v", err)
	}
	port := tmpLn.Addr().(*net.TCPAddr).Port
	_ = tmpLn.Close()

	srv := httptest.NewServer(NewHandler(Config{
		TCPPortRangeStart: port,
		TCPPortRangeEnd:   port,
	}, Dependencies{
		TokenValidator: store,
		TokenResolver:  store,
		Reservations:   store,
	}))
	t.Cleanup(srv.Close)

	t.Run("owner can claim and reclaim", func(t *testing.T) {
		ws, session := dialTestControl(t, srv.URL)
		defer func() {
			_ = session.Close()
			_ = ws.Close(websocket.StatusNormalClosure, "closed")
		}()

		stream, err := session.OpenStream()
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}
		defer stream.Close()

		if err := json.NewEncoder(stream).Encode(control.CreateTCPTunnelRequest{
			Type:       "tcp",
			Authtoken:  tokenA,
			RemotePort: port,
		}); err != nil {
			t.Fatalf("encode: %v", err)
		}

		var resp control.CreateTCPTunnelResponse
		if err := json.NewDecoder(stream).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Error != "" {
			t.Fatalf("error = %q, want empty", resp.Error)
		}
		if resp.RemotePort != port {
			t.Fatalf("remote port = %d, want %d", resp.RemotePort, port)
		}
	})

	t.Run("other token rejected", func(t *testing.T) {
		ws, session := dialTestControl(t, srv.URL)
		defer func() {
			_ = session.Close()
			_ = ws.Close(websocket.StatusNormalClosure, "closed")
		}()

		stream, err := session.OpenStream()
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}
		defer stream.Close()

		if err := json.NewEncoder(stream).Encode(control.CreateTCPTunnelRequest{
			Type:       "tcp",
			Authtoken:  tokenB,
			RemotePort: port,
		}); err != nil {
			t.Fatalf("encode: %v", err)
		}

		var resp control.CreateTCPTunnelResponse
		if err := json.NewDecoder(stream).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Error != "unauthorized" {
			t.Fatalf("error = %q, want %q", resp.Error, "unauthorized")
		}
	})

	t.Run("owner can reclaim after denial", func(t *testing.T) {
		ws, session := dialTestControl(t, srv.URL)
		defer func() {
			_ = session.Close()
			_ = ws.Close(websocket.StatusNormalClosure, "closed")
		}()

		stream, err := session.OpenStream()
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}
		defer stream.Close()

		if err := json.NewEncoder(stream).Encode(control.CreateTCPTunnelRequest{
			Type:       "tcp",
			Authtoken:  tokenA,
			RemotePort: port,
		}); err != nil {
			t.Fatalf("encode: %v", err)
		}

		var resp control.CreateTCPTunnelResponse
		if err := json.NewDecoder(stream).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Error != "" {
			t.Fatalf("error = %q, want empty", resp.Error)
		}
		if resp.RemotePort != port {
			t.Fatalf("remote port = %d, want %d", resp.RemotePort, port)
		}
	})
}
