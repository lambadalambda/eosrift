//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
)

func TestHTTPTunnel_HTTPSUpstream(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("hello-from-https-upstream\n"))
	}))
	t.Cleanup(upstream.Close)

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL(), u.Host, client.HTTPTunnelOptions{
		Authtoken:             getenv("EOSRIFT_AUTHTOKEN", ""),
		UpstreamScheme:        "https",
		UpstreamTLSSkipVerify: true,
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
	if got, want := string(body), "hello-from-https-upstream\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
