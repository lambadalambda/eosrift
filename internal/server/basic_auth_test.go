package server

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type recordingSession struct {
	openCount atomic.Int32
	authCh    chan string
}

func (s *recordingSession) OpenStream() (net.Conn, error) {
	s.openCount.Add(1)

	a, b := net.Pipe()

	go func() {
		defer b.Close()

		req, err := http.ReadRequest(bufio.NewReader(b))
		if err != nil {
			return
		}

		if s.authCh != nil {
			s.authCh <- req.Header.Get("Authorization")
		}

		body := "ok\n"
		_, _ = fmt.Fprintf(b, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
	}()

	return a, nil
}

func (s *recordingSession) Close() error { return nil }

func TestHTTPTunnel_BasicAuth(t *testing.T) {
	t.Parallel()

	registry := NewTunnelRegistry()
	sess := &recordingSession{authCh: make(chan string, 1)}

	if err := registry.RegisterHTTPTunnel("abcd1234", sess, &basicAuthCredential{Username: "user", Password: "pass"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test"}, registry)

	t.Run("missing auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusUnauthorized, rr.Body.String())
		}
		if rr.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("missing WWW-Authenticate header")
		}
		if sess.openCount.Load() != 0 {
			t.Fatalf("open count = %d, want 0", sess.openCount.Load())
		}
	})

	t.Run("valid auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"
		req.SetBasicAuth("user", "pass")

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if rr.Body.String() != "ok\n" {
			t.Fatalf("body = %q, want %q", rr.Body.String(), "ok\n")
		}

		select {
		case got := <-sess.authCh:
			if got != "" {
				t.Fatalf("upstream Authorization = %q, want empty", got)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timeout waiting for upstream request")
		}
	})
}
