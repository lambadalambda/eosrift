package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandler_healthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	NewHandler(Config{}, Dependencies{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got, want := rec.Body.String(), "ok\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestNewHandler_landingPage_baseDomainIndex(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
	}, Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/", nil)
	req.Host = "eosrift.com"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("content-type = %q, want text/html", got)
	}

	if got := rec.Body.String(); !strings.Contains(got, "Self-Hosted Tunnel Service") {
		t.Fatalf("body missing marker (len=%d)", len(got))
	}
}

func TestNewHandler_landingPage_baseDomainStyle(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
	}, Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/style.css", nil)
	req.Host = "eosrift.com"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("content-type = %q, want text/css", got)
	}

	if got := rec.Body.String(); !strings.Contains(got, "--bg-primary") {
		t.Fatalf("body missing marker (len=%d)", len(got))
	}
}

func TestNewHandler_landingPage_notServedOnTunnelHost(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
	}, Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "http://abcd1234.tunnel.eosrift.com/style.css", nil)
	req.Host = "abcd1234.tunnel.eosrift.com"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestNewHandler_caddyAsk_allowsBaseAndTunnelDomains(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
	}, Dependencies{})

	cases := []struct {
		name       string
		domain     string
		wantStatus int
	}{
		{"base domain", "eosrift.com", http.StatusOK},
		{"base domain (case)", "EOSRIFT.COM", http.StatusOK},
		{"tunnel domain apex", "tunnel.eosrift.com", http.StatusOK},
		{"tunnel subdomain", "abcd1234.tunnel.eosrift.com", http.StatusOK},
		{"other subdomain rejected", "www.eosrift.com", http.StatusForbidden},
		{"other domain rejected", "example.com", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/caddy/ask?domain="+tc.domain, nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
