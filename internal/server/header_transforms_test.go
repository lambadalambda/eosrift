package server

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type captureSession struct {
	gotCh chan capturedHeaders
}

type capturedHeaders struct {
	Remove   string
	Add      string
	Override string
}

func (s *captureSession) OpenStream() (net.Conn, error) {
	a, b := net.Pipe()

	go func() {
		defer b.Close()

		req, err := http.ReadRequest(bufio.NewReader(b))
		if err != nil {
			return
		}
		_ = req.Body.Close()

		if s.gotCh != nil {
			s.gotCh <- capturedHeaders{
				Remove:   req.Header.Get("X-Remove"),
				Add:      req.Header.Get("X-Add"),
				Override: req.Header.Get("X-Override"),
			}
		}

		body := "ok\n"
		_, _ = fmt.Fprintf(b, "HTTP/1.1 200 OK\r\nX-Upstream: yes\r\nX-Keep: 1\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
	}()

	return a, nil
}

func (s *captureSession) Close() error { return nil }

func TestHTTPTunnel_HeaderTransforms(t *testing.T) {
	t.Parallel()

	gotCh := make(chan capturedHeaders, 1)

	registry := NewTunnelRegistry()
	sess := &captureSession{gotCh: gotCh}

	if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
		RequestHeaderRemove: []string{"X-Remove"},
		RequestHeaderAdd: []headerKV{
			{Name: "X-Add", Value: "yes"},
			{Name: "X-Override", Value: "new"},
		},
		ResponseHeaderRemove: []string{"X-Upstream"},
		ResponseHeaderAdd: []headerKV{
			{Name: "X-Edge", Value: "ok"},
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test"}, registry)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
	req.Host = "abcd1234.tunnel.eosrift.test"
	req.Header.Set("X-Remove", "bye")
	req.Header.Set("X-Override", "old")

	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want %q", rr.Body.String(), "ok\n")
	}
	if got, want := rr.Header().Get("X-Upstream"), ""; got != want {
		t.Fatalf("X-Upstream = %q, want %q", got, want)
	}
	if got, want := rr.Header().Get("X-Edge"), "ok"; got != want {
		t.Fatalf("X-Edge = %q, want %q", got, want)
	}
	if got, want := rr.Header().Get("X-Keep"), "1"; got != want {
		t.Fatalf("X-Keep = %q, want %q", got, want)
	}

	select {
	case got := <-gotCh:
		if got.Remove != "" {
			t.Fatalf("upstream X-Remove = %q, want empty", got.Remove)
		}
		if got.Add != "yes" {
			t.Fatalf("upstream X-Add = %q, want %q", got.Add, "yes")
		}
		if got.Override != "new" {
			t.Fatalf("upstream X-Override = %q, want %q", got.Override, "new")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for upstream request")
	}
}
