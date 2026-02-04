//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
	"nhooyr.io/websocket"
)

func TestHTTPTunnel_WebSocketEcho(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "closed")

		ctx := r.Context()
		for {
			msgType, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if err := conn.Write(ctx, msgType, msg); err != nil {
				return
			}
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{
		Handler:           upstream,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(tunnelCtx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
		Authtoken: getenv("EOSRIFT_AUTHTOKEN", ""),
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	host := fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()

	ws, _, err := websocket.Dial(dialCtx, wsURL("/ws"), &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
		Host:            host,
	})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "closed")

	msgCtx, msgCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer msgCancel()

	want := []byte("hello-ws")
	if err := ws.Write(msgCtx, websocket.MessageText, want); err != nil {
		t.Fatalf("ws write: %v", err)
	}

	_, got, err := ws.Read(msgCtx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("echo = %q, want %q", string(got), string(want))
	}
}
