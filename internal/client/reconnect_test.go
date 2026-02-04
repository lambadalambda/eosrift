package client

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/config"
	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/mux"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func TestHTTPTunnel_Reconnect_ResumesAssignedDomain(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	echoLn := startTestEchoListener(t)
	defer echoLn.Close()

	const (
		wantID  = "abcd1234"
		wantURL = "https://abcd1234.tunnel.eosrift.test"
	)

	reqCh := make(chan control.CreateHTTPTunnelRequest, 2)
	disconnect1 := make(chan struct{})
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/control" {
			http.NotFound(w, r)
			return
		}

		attempt := attempts.Add(1)

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "closed")

		netConn := websocket.NetConn(r.Context(), conn, websocket.MessageBinary)
		session, err := yamux.Server(netConn, mux.QuietYamuxConfig())
		if err != nil {
			return
		}
		defer session.Close()

		ctrlStream, err := session.AcceptStream()
		if err != nil {
			return
		}

		var req control.CreateHTTPTunnelRequest
		if err := json.NewDecoder(ctrlStream).Decode(&req); err != nil {
			return
		}
		reqCh <- req

		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type: "http",
			ID:   wantID,
			URL:  wantURL,
		})
		_ = ctrlStream.Close()

		if attempt == 1 {
			select {
			case <-disconnect1:
				_ = conn.Close(websocket.StatusGoingAway, "reconnect")
				return
			case <-r.Context().Done():
				return
			}
		}

		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	controlURL, err := config.ControlURLFromServerAddr(srv.URL)
	if err != nil {
		t.Fatalf("control url: %v", err)
	}

	tunnel, err := StartHTTPTunnelWithOptions(ctx, controlURL, echoLn.Addr().String(), HTTPTunnelOptions{
		Authtoken: "tok_123",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	if tunnel.ID != wantID || tunnel.URL != wantURL {
		t.Fatalf("tunnel id/url = %q/%q, want %q/%q", tunnel.ID, tunnel.URL, wantID, wantURL)
	}

	req1 := recvWithTimeout(t, ctx, reqCh)
	if req1.Domain != "" || req1.Subdomain != "" {
		t.Fatalf("req1 domain/subdomain = %q/%q, want empty", req1.Domain, req1.Subdomain)
	}

	close(disconnect1)

	req2 := recvWithTimeout(t, ctx, reqCh)
	if req2.Subdomain != "" {
		t.Fatalf("req2 subdomain = %q, want empty", req2.Subdomain)
	}
	if got, want := strings.TrimSpace(req2.Domain), "abcd1234.tunnel.eosrift.test"; got != want {
		t.Fatalf("req2 domain = %q, want %q", got, want)
	}

	_ = tunnel.Close()
	waitDone(t, ctx, tunnel.Wait)
}

func TestTCPTunnel_Reconnect_ResumesPort(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	echoLn := startTestEchoListener(t)
	defer echoLn.Close()

	const wantPort = 20001

	reqCh := make(chan control.CreateTCPTunnelRequest, 2)
	disconnect1 := make(chan struct{})
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/control" {
			http.NotFound(w, r)
			return
		}

		attempt := attempts.Add(1)

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "closed")

		netConn := websocket.NetConn(r.Context(), conn, websocket.MessageBinary)
		session, err := yamux.Server(netConn, mux.QuietYamuxConfig())
		if err != nil {
			return
		}
		defer session.Close()

		ctrlStream, err := session.AcceptStream()
		if err != nil {
			return
		}

		var req control.CreateTCPTunnelRequest
		if err := json.NewDecoder(ctrlStream).Decode(&req); err != nil {
			return
		}
		reqCh <- req

		_ = json.NewEncoder(ctrlStream).Encode(control.CreateTCPTunnelResponse{
			Type:       "tcp",
			RemotePort: wantPort,
		})
		_ = ctrlStream.Close()

		if attempt == 1 {
			select {
			case <-disconnect1:
				_ = conn.Close(websocket.StatusGoingAway, "reconnect")
				return
			case <-r.Context().Done():
				return
			}
		}

		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	controlURL, err := config.ControlURLFromServerAddr(srv.URL)
	if err != nil {
		t.Fatalf("control url: %v", err)
	}

	tunnel, err := StartTCPTunnelWithOptions(ctx, controlURL, echoLn.Addr().String(), TCPTunnelOptions{
		Authtoken: "tok_123",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	if tunnel.RemotePort != wantPort {
		t.Fatalf("remote port = %d, want %d", tunnel.RemotePort, wantPort)
	}

	req1 := recvWithTimeout(t, ctx, reqCh)
	if req1.RemotePort != 0 {
		t.Fatalf("req1 remote port = %d, want %d", req1.RemotePort, 0)
	}

	close(disconnect1)

	req2 := recvWithTimeout(t, ctx, reqCh)
	if req2.RemotePort != wantPort {
		t.Fatalf("req2 remote port = %d, want %d", req2.RemotePort, wantPort)
	}

	_ = tunnel.Close()
	waitDone(t, ctx, tunnel.Wait)
}

func startTestEchoListener(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	return ln
}

func recvWithTimeout[T any](t *testing.T, ctx context.Context, ch <-chan T) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatalf("timeout waiting for value: %v", ctx.Err())
		var zero T
		return zero
	}
}

func waitDone(t *testing.T, ctx context.Context, wait func() error) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- wait() }()

	select {
	case <-ctx.Done():
		t.Fatalf("timeout waiting for tunnel to stop: %v", ctx.Err())
	case <-done:
	}
}
