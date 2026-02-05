package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/mux"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func TestControlAuth_HTTP_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain: "tunnel.example.com",
	}, Dependencies{
		TokenValidator: staticValidator{token: "secret"},
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
		Authtoken: "wrong",
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

func TestControlAuth_HTTP_Authorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewHandler(Config{
		TunnelDomain: "tunnel.example.com",
	}, Dependencies{
		TokenValidator: staticValidator{token: "secret"},
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
		Authtoken: "secret",
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
	if resp.ID == "" || resp.URL == "" {
		t.Fatalf("id/url = %q/%q, want non-empty", resp.ID, resp.URL)
	}
}

func dialTestControl(t *testing.T, httpBaseURL string) (*websocket.Conn, *yamux.Session) {
	t.Helper()

	u, err := url.Parse(httpBaseURL)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/control"

	ctx := context.Background()
	ws, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}

	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	session, err := yamux.Client(netConn, mux.QuietYamuxConfig())
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "yamux error")
		t.Fatalf("yamux client: %v", err)
	}

	return ws, session
}

type staticValidator struct {
	token string
}

func (v staticValidator) ValidateToken(ctx context.Context, token string) (bool, error) {
	return token == v.token, nil
}
