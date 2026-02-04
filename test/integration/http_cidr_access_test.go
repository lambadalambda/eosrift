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

func TestHTTPTunnel_CIDRAccessControl(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello\n"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Handler: upstream}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}

	t.Run("allowlist permits docker network clients", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
			Authtoken:   getenv("EOSRIFT_AUTHTOKEN", ""),
			AllowCIDRs:  []string{testNetworkCIDR()},
			HostHeader:  "preserve",
		})
		if err != nil {
			t.Fatalf("start http tunnel: %v", err)
		}
		defer tunnel.Close()

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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, string(body))
		}
	})

	t.Run("denylist blocks docker network clients", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
			Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
			DenyCIDRs:  []string{testNetworkCIDR()},
			HostHeader: "preserve",
		})
		if err != nil {
			t.Fatalf("start http tunnel: %v", err)
		}
		defer tunnel.Close()

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

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
		}
	})

	t.Run("untrusted xff does not bypass allowlist", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
			Authtoken:   getenv("EOSRIFT_AUTHTOKEN", ""),
			AllowCIDRs:  []string{"1.2.3.4/32"},
			HostHeader:  "preserve",
		})
		if err != nil {
			t.Fatalf("start http tunnel: %v", err)
		}
		defer tunnel.Close()

		req, err := http.NewRequest(http.MethodGet, httpURL("/hello"), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)
		req.Header.Set("X-Forwarded-For", "1.2.3.4")

		resp, err := clientHTTP.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
		}
	})
}
