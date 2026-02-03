//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/inspect"
)

func TestInspector_CapturesHTTPTunnel(t *testing.T) {
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

	store := inspect.NewStore(inspect.StoreConfig{MaxEntries: 50})
	inspector := httptest.NewServer(inspect.Handler(store))
	t.Cleanup(inspector.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, "ws://server:8080/control", ln.Addr().String(), client.HTTPTunnelOptions{
		Inspector: store,
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
	_, _ = io.ReadAll(resp.Body)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(inspector.URL + "/api/requests")
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		var body struct {
			Requests []inspect.Entry `json:"requests"`
		}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			t.Fatalf("decode inspector response: %v (body=%q)", err, string(bodyBytes))
		}

		if len(body.Requests) == 0 {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		got := body.Requests[0]
		if got.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", got.Method, http.MethodGet)
		}
		if got.Path != "/hello" {
			t.Fatalf("path = %q, want %q", got.Path, "/hello")
		}
		if got.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", got.StatusCode, http.StatusOK)
		}
		if got.TunnelID != tunnel.ID {
			t.Fatalf("tunnel_id = %q, want %q", got.TunnelID, tunnel.ID)
		}

		return
	}

	t.Fatalf("inspector did not record a request before deadline")
}

