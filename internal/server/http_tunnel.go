package server

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
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

		entry, ok := registry.GetHTTPTunnel(id)
		if !ok {
			http.NotFound(w, r)
			return
		}

		if len(entry.allowCIDRs) > 0 || len(entry.denyCIDRs) > 0 {
			ip, ok := requestClientIP(r, cfg.TrustProxyHeaders)
			if !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if cidrListContains(entry.denyCIDRs, ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if len(entry.allowCIDRs) > 0 && !cidrListContains(entry.allowCIDRs, ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if entry.basicAuth != nil {
			user, pass, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(user), []byte(entry.basicAuth.Username)) != 1 ||
				subtle.ConstantTimeCompare([]byte(pass), []byte(entry.basicAuth.Password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="EosRift"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			r.Header.Del("Authorization")
		}

		if !cfg.TrustProxyHeaders {
			stripForwardedHeaders(r.Header)
		}

		transport := &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         func(ctx context.Context, network, addr string) (net.Conn, error) { return entry.session.OpenStream() },
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

func requestClientIP(r *http.Request, trustProxyHeaders bool) (netip.Addr, bool) {
	if trustProxyHeaders {
		if v := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); v != "" {
			first := strings.TrimSpace(strings.Split(v, ",")[0])
			if ip, ok := parseNetipAddr(first); ok {
				return ip, true
			}
		}
		if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
			if ip, ok := parseNetipAddr(v); ok {
				return ip, true
			}
		}
	}

	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return netip.Addr{}, false
	}

	// Best-effort host:port stripping.
	if strings.Contains(host, ":") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}

	return parseNetipAddr(host)
}

func parseNetipAddr(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

func cidrListContains(prefixes []netip.Prefix, ip netip.Addr) bool {
	ip = ip.Unmap()
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
