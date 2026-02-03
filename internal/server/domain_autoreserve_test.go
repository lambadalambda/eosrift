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

type createHTTPWithDomainRequest struct {
	Type      string `json:"type"`
	Authtoken string `json:"authtoken,omitempty"`
	Domain    string `json:"domain,omitempty"`
}

func TestControlHTTP_Domain_AutoReservesOnFirstUse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := auth.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	owner, ownerToken, err := store.CreateToken(ctx, "owner")
	if err != nil {
		t.Fatalf("create token owner: %v", err)
	}
	_, otherToken, err := store.CreateToken(ctx, "other")
	if err != nil {
		t.Fatalf("create token other: %v", err)
	}

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain: "tunnel.example.com",
	}, Dependencies{
		TokenValidator: store,
		TokenResolver:  store,
		Reservations:   store,
	}))
	t.Cleanup(srv.Close)

	// First use by owner should succeed and reserve.
	{
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

		if err := json.NewEncoder(stream).Encode(createHTTPWithDomainRequest{
			Type:      "http",
			Authtoken: ownerToken,
			Domain:    "demo.tunnel.example.com",
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

	gotID, ok, err := store.ReservedSubdomainTokenID(ctx, "demo")
	if err != nil {
		t.Fatalf("lookup reserved: %v", err)
	}
	if !ok {
		t.Fatalf("reserved ok = false, want true")
	}
	if gotID != owner.ID {
		t.Fatalf("reserved token id = %d, want %d", gotID, owner.ID)
	}

	// Second use by a different token should be rejected.
	{
		ws, session := dialTestControl(t, srv.URL)
		defer func() {
			_ = session.Close()
			_ = ws.Close(websocket.StatusNormalClosure, "closed")
		}()

		stream, err := session.OpenStream()
		if err != nil {
			t.Fatalf("open stream2: %v", err)
		}
		defer stream.Close()

		if err := json.NewEncoder(stream).Encode(createHTTPWithDomainRequest{
			Type:      "http",
			Authtoken: otherToken,
			Domain:    "demo.tunnel.example.com",
		}); err != nil {
			t.Fatalf("encode2: %v", err)
		}

		var resp control.CreateHTTPTunnelResponse
		if err := json.NewDecoder(stream).Decode(&resp); err != nil {
			t.Fatalf("decode2: %v", err)
		}

		if resp.Error == "" {
			t.Fatalf("error = %q, want non-empty", resp.Error)
		}
	}
}

func TestControlHTTP_Domain_RejectsNonTunnelDomain(t *testing.T) {
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

	if err := json.NewEncoder(stream).Encode(createHTTPWithDomainRequest{
		Type:      "http",
		Authtoken: token,
		Domain:    "demo.example.com",
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

