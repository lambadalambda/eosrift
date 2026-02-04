package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandler_metrics_disabledByDefault(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
	}, Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/metrics", nil)
	req.Host = "eosrift.com"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestNewHandler_metrics_requiresToken(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:        "eosrift.com",
		TunnelDomain:      "tunnel.eosrift.com",
		MetricsToken:      "secret",
		TCPPortRangeStart: 20000,
		TCPPortRangeEnd:   20010,
	}, Dependencies{})

	t.Run("no token", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/metrics", nil)
		req.Host = "eosrift.com"
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("with token", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/metrics", nil)
		req.Host = "eosrift.com"
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
			t.Fatalf("content-type = %q, want text/plain", got)
		}
		if body := rec.Body.String(); !strings.Contains(body, "eosrift_active_http_tunnels") {
			t.Fatalf("body missing marker (len=%d)", len(body))
		}
	})

	t.Run("not on tunnel host", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://abcd1234.tunnel.eosrift.com/metrics", nil)
		req.Host = "abcd1234.tunnel.eosrift.com"
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}
