package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestHTTPTunnel_IPAccessControl(t *testing.T) {
	t.Parallel()

	t.Run("allowlist", func(t *testing.T) {
		t.Parallel()

		registry := NewTunnelRegistry()
		sess := &recordingSession{}

		allow := []netip.Prefix{netip.MustParsePrefix("1.2.3.0/24")}
		if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
			AllowCIDRs: allow,
		}); err != nil {
			t.Fatalf("register: %v", err)
		}

		h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test"}, registry)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"
		req.RemoteAddr = "1.2.3.4:1234"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
		}
	})

	t.Run("denylist", func(t *testing.T) {
		t.Parallel()

		registry := NewTunnelRegistry()
		sess := &recordingSession{}

		deny := []netip.Prefix{netip.MustParsePrefix("1.2.3.4/32")}
		if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
			DenyCIDRs: deny,
		}); err != nil {
			t.Fatalf("register: %v", err)
		}

		h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test"}, registry)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"
		req.RemoteAddr = "1.2.3.4:1234"

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusForbidden, rr.Body.String())
		}
	})

	t.Run("trust proxy headers disabled: xff does not bypass allowlist", func(t *testing.T) {
		t.Parallel()

		registry := NewTunnelRegistry()
		sess := &recordingSession{}

		allow := []netip.Prefix{netip.MustParsePrefix("1.2.3.4/32")}
		if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
			AllowCIDRs: allow,
		}); err != nil {
			t.Fatalf("register: %v", err)
		}

		h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test", TrustProxyHeaders: false}, registry)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"
		req.RemoteAddr = "5.6.7.8:1234"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusForbidden, rr.Body.String())
		}
	})

	t.Run("trust proxy headers enabled: xff applies", func(t *testing.T) {
		t.Parallel()

		registry := NewTunnelRegistry()
		sess := &recordingSession{}

		allow := []netip.Prefix{netip.MustParsePrefix("1.2.3.4/32")}
		if err := registry.RegisterHTTPTunnel("abcd1234", sess, httpTunnelOptions{
			AllowCIDRs: allow,
		}); err != nil {
			t.Fatalf("register: %v", err)
		}

		h := httpTunnelProxyHandler(Config{TunnelDomain: "tunnel.eosrift.test", TrustProxyHeaders: true}, registry)

		req := httptest.NewRequest(http.MethodGet, "http://example.test/hello", nil)
		req.Host = "abcd1234.tunnel.eosrift.test"
		req.RemoteAddr = "5.6.7.8:1234"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")

		rr := httptest.NewRecorder()
		h(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%q)", rr.Code, http.StatusOK, rr.Body.String())
		}
	})
}
