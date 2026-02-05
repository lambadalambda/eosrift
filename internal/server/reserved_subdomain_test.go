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

func TestControlHTTP_ReservedSubdomain_AllowsOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := auth.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	rec, token, err := store.CreateToken(ctx, "owner")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := store.ReserveSubdomain(ctx, rec.ID, "demo"); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain: "tunnel.example.com",
	}, Dependencies{
		TokenValidator: store,
		TokenResolver:  store,
		Reservations:   store,
	}))
	t.Cleanup(srv.Close)

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

	if err := control.WriteJSON(stream, control.CreateHTTPTunnelRequest{
		Type:      "http",
		Authtoken: token,
		Subdomain: "demo",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var resp control.CreateHTTPTunnelResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Error != "" {
		t.Fatalf("error = %q, want empty", resp.Error)
	}
	if resp.ID != "demo" {
		t.Fatalf("id = %q, want %q", resp.ID, "demo")
	}
	if resp.URL != "https://demo.tunnel.example.com" {
		t.Fatalf("url = %q, want %q", resp.URL, "https://demo.tunnel.example.com")
	}
}

func TestControlHTTP_ReservedSubdomain_RejectsOtherToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := auth.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	owner, _, err := store.CreateToken(ctx, "owner")
	if err != nil {
		t.Fatalf("create token owner: %v", err)
	}
	_, otherToken, err := store.CreateToken(ctx, "other")
	if err != nil {
		t.Fatalf("create token other: %v", err)
	}

	if err := store.ReserveSubdomain(ctx, owner.ID, "demo"); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain: "tunnel.example.com",
	}, Dependencies{
		TokenValidator: store,
		TokenResolver:  store,
		Reservations:   store,
	}))
	t.Cleanup(srv.Close)

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

	if err := control.WriteJSON(stream, control.CreateHTTPTunnelRequest{
		Type:      "http",
		Authtoken: otherToken,
		Subdomain: "demo",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var resp control.CreateHTTPTunnelResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("error = %q, want non-empty", resp.Error)
	}
}
