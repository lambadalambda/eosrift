//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
)

func TestHTTPTunnel_Basic(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello-from-upstream\n"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Handler: upstream}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, "ws://server:8080/control", ln.Addr().String(), client.HTTPTunnelOptions{
		Authtoken: getenv("EOSRIFT_AUTHTOKEN", ""),
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	req, err := http.NewRequest(http.MethodGet, "http://server:8080/hello", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)

	clientHTTP := &http.Client{Timeout: 5 * time.Second}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if got, want := string(body), "hello-from-upstream\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
