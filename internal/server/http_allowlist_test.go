package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTunnel_MethodAndPathAllowlist(t *testing.T) {
	t.Parallel()

	registry := NewTunnelRegistry()
	sess := &recordingSession{}

	if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
		AllowMethods:      []string{"GET"},
		AllowPaths:        []string{"/allowed"},
		AllowPathPrefixes: []string{"/api/"},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test"}, registry)

	t.Run("denies disallowed method", func(t *testing.T) {
		sess.openCount.Store(0)

		req := httptest.NewRequest(http.MethodPost, "http://example.test/allowed", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
		if sess.openCount.Load() != 0 {
			t.Fatalf("open count = %d, want 0", sess.openCount.Load())
		}
	})

	t.Run("denies disallowed path", func(t *testing.T) {
		sess.openCount.Store(0)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/nope", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
		if sess.openCount.Load() != 0 {
			t.Fatalf("open count = %d, want 0", sess.openCount.Load())
		}
	})

	t.Run("allows exact path", func(t *testing.T) {
		sess.openCount.Store(0)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/allowed", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if rr.Body.String() != "ok\n" {
			t.Fatalf("body = %q, want %q", rr.Body.String(), "ok\n")
		}
		if sess.openCount.Load() != 1 {
			t.Fatalf("open count = %d, want 1", sess.openCount.Load())
		}
	})

	t.Run("allows path prefix", func(t *testing.T) {
		sess.openCount.Store(0)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/api/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if rr.Body.String() != "ok\n" {
			t.Fatalf("body = %q, want %q", rr.Body.String(), "ok\n")
		}
		if sess.openCount.Load() != 1 {
			t.Fatalf("open count = %d, want 1", sess.openCount.Load())
		}
	})
}
