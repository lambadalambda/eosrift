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

func TestHTTPTunnel_HeaderTransforms(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "present")
		_, _ = fmt.Fprintf(w, "added=%s removed=%s\n", r.Header.Get("X-Added"), r.Header.Get("X-Remove"))
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
		Authtoken: getenv("EOSRIFT_AUTHTOKEN", ""),
		RequestHeaderAdd: []client.HeaderKV{
			{Name: "X-Added", Value: "yes"},
		},
		RequestHeaderRemove: []string{"X-Remove"},
		ResponseHeaderAdd: []client.HeaderKV{
			{Name: "X-Edge", Value: "ok"},
		},
		ResponseHeaderRemove: []string{"X-Upstream"},
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	req, err := http.NewRequest(http.MethodGet, httpURL("/headers"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)
	req.Header.Set("X-Remove", "bye")

	clientHTTP := &http.Client{Timeout: 5 * time.Second}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got, want := string(body), "added=yes removed=\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	if got, want := resp.Header.Get("X-Upstream"), ""; got != want {
		t.Fatalf("X-Upstream = %q, want %q", got, want)
	}
	if got, want := resp.Header.Get("X-Edge"), "ok"; got != want {
		t.Fatalf("X-Edge = %q, want %q", got, want)
	}
}

