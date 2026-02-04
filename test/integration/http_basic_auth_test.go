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

func TestHTTPTunnel_BasicAuth(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream-Auth", r.Header.Get("Authorization"))
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

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
		Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
		BasicAuth:  "user:pass",
		HostHeader: "preserve",
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}

	t.Run("unauthorized", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, httpURL("/hello"), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)

		resp, err := clientHTTP.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
		if resp.Header.Get("WWW-Authenticate") == "" {
			t.Fatalf("missing WWW-Authenticate header")
		}
	})

	t.Run("authorized", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, httpURL("/hello"), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)
		req.SetBasicAuth("user", "pass")

		resp, err := clientHTTP.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, string(body))
		}
		if got := resp.Header.Get("X-Upstream-Auth"); got != "" {
			t.Fatalf("upstream auth header = %q, want empty", got)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if got, want := string(body), "hello-from-upstream\n"; got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})
}
