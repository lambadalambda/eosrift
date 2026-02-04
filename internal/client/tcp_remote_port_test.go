package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/config"
	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/mux"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func TestTCPTunnel_RequestRemotePort_SendsRemotePort(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	echoLn := startTestEchoListener(t)
	t.Cleanup(func() { _ = echoLn.Close() })

	const wantPort = 20005

	reqCh := make(chan control.CreateTCPTunnelRequest, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/control" {
			http.NotFound(w, r)
			return
		}

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

		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	controlURL, err := config.ControlURLFromServerAddr(srv.URL)
	if err != nil {
		t.Fatalf("control url: %v", err)
	}

	tunnel, err := StartTCPTunnelWithOptions(ctx, controlURL, echoLn.Addr().String(), TCPTunnelOptions{
		Authtoken:  "tok_123",
		RemotePort: wantPort,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	if tunnel.RemotePort != wantPort {
		t.Fatalf("remote port = %d, want %d", tunnel.RemotePort, wantPort)
	}

	req := recvWithTimeout(t, ctx, reqCh)
	if req.RemotePort != wantPort {
		t.Fatalf("requested remote port = %d, want %d", req.RemotePort, wantPort)
	}

	_ = tunnel.Close()
	waitDone(t, ctx, tunnel.Wait)
}
