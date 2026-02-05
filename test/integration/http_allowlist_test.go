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

func TestHTTPTunnel_MethodAndPathAllowlist(t *testing.T) {
	t.Parallel()

	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
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
		Authtoken:         getenv("EOSRIFT_AUTHTOKEN", ""),
		AllowMethods:      []string{"GET"},
		AllowPaths:        []string{"/allowed"},
		AllowPathPrefixes: []string{"/api/"},
		HostHeader:        "preserve",
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path string) (*http.Response, string) {
		req, err := http.NewRequest(method, httpURL(path), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)

		resp, err := clientHTTP.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp, string(body)
	}

	t.Run("denies disallowed method", func(t *testing.T) {
		resp, body := do(http.MethodPost, "/allowed")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusNotFound, body)
		}
	})

	t.Run("denies disallowed path", func(t *testing.T) {
		resp, body := do(http.MethodGet, "/nope")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusNotFound, body)
		}
	})

	t.Run("allows exact path", func(t *testing.T) {
		resp, body := do(http.MethodGet, "/allowed")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, body)
		}
		if body != "ok\n" {
			t.Fatalf("body = %q, want %q", body, "ok\n")
		}
	})

	t.Run("allows path prefix", func(t *testing.T) {
		resp, body := do(http.MethodGet, "/api/hello")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, body)
		}
		if body != "ok\n" {
			t.Fatalf("body = %q, want %q", body, "ok\n")
		}
	})
}

