package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func httpTunnelProxyHandler(cfg Config, registry *TunnelRegistry) http.HandlerFunc {
	target := &url.URL{
		Scheme: "http",
		Host:   "upstream",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := tunnelIDFromHost(r.Host, cfg.TunnelDomain)
		if !ok {
			http.NotFound(w, r)
			return
		}

		session, ok := registry.GetHTTPTunnel(id)
		if !ok {
			http.NotFound(w, r)
			return
		}

		if !cfg.TrustProxyHeaders {
			stripForwardedHeaders(r.Header)
		}

		transport := &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         func(ctx context.Context, network, addr string) (net.Conn, error) { return session.OpenStream() },
			ForceAttemptHTTP2:   false,
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: -1,
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				// Preserve original host (ngrok-like) but send the request over a
				// tunneled TCP stream to the local upstream.
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
			},
			Transport: transport,
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				http.Error(rw, "bad gateway", http.StatusBadGateway)
			},
		}

		// Ensure we don't keep resources around for this transport.
		defer transport.CloseIdleConnections()

		proxy.ServeHTTP(w, r)
	}
}

func stripForwardedHeaders(h http.Header) {
	// Avoid trusting proxy-provided headers from untrusted clients.
	h.Del("Forwarded")
	h.Del("X-Forwarded-For")
	h.Del("X-Forwarded-Host")
	h.Del("X-Forwarded-Proto")
	h.Del("X-Forwarded-Port")
	h.Del("X-Real-IP")
}

func tunnelIDFromHost(host, tunnelDomain string) (string, bool) {
	h := normalizeDomain(host)
	td := normalizeDomain(tunnelDomain)
	if td == "" {
		return "", false
	}

	suffix := "." + td
	if !strings.HasSuffix(h, suffix) {
		return "", false
	}

	prefix := strings.TrimSuffix(h, suffix)
	if prefix == "" {
		return "", false
	}
	if strings.Contains(prefix, ".") {
		return "", false
	}
	return prefix, true
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimSuffix(domain, ".")
	domain = strings.ToLower(domain)

	// Best-effort host:port stripping.
	if strings.Contains(domain, ":") {
		if host, _, err := net.SplitHostPort(domain); err == nil {
			domain = host
		}
	}
	return domain
}
