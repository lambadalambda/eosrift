package server

import (
	"context"
	"crypto/subtle"
	"errors"
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

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			entry, ok := tunnelEntryFromContext(ctx)
			if !ok || entry.session == nil {
				return nil, errors.New("missing tunnel session")
			}
			return entry.session.OpenStream()
		},
		ForceAttemptHTTP2:   false,
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: -1,
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Preserve original host (ngrok-like) but send the request over a
			// tunneled TCP stream to the local upstream.
			pr.SetURL(target)
			pr.Out.Host = pr.In.Host

			if cfg.TrustProxyHeaders {
				copyProxyForwardedHeaders(pr.Out.Header, pr.In.Header)
			} else {
				stripForwardedHeaders(pr.Out.Header)
				pr.SetXForwarded()
			}

			entry, ok := tunnelEntryFromContext(pr.In.Context())
			if ok {
				applyHeaderTransforms(pr.Out.Header, entry.requestHeaderRemove, entry.requestHeaderAdd)
			}
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			if resp == nil || resp.Request == nil {
				return nil
			}

			entry, ok := tunnelEntryFromContext(resp.Request.Context())
			if ok {
				applyHeaderTransforms(resp.Header, entry.responseHeaderRemove, entry.responseHeaderAdd)
			}
			return nil
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			http.Error(rw, "bad gateway", http.StatusBadGateway)
		},
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

		if len(entry.allowMethods) > 0 {
			method := strings.ToUpper(strings.TrimSpace(r.Method))
			allowed := false
			for _, m := range entry.allowMethods {
				if method == strings.ToUpper(strings.TrimSpace(m)) {
					allowed = true
					break
				}
			}
			if !allowed {
				http.NotFound(w, r)
				return
			}
		}

		if len(entry.allowPaths) > 0 || len(entry.allowPathPrefixes) > 0 {
			path := r.URL.Path
			allowed := false
			for _, p := range entry.allowPaths {
				if path == strings.TrimSpace(p) {
					allowed = true
					break
				}
			}
			if !allowed {
				for _, p := range entry.allowPathPrefixes {
					p = strings.TrimSpace(p)
					if p != "" && strings.HasPrefix(path, p) {
						allowed = true
						break
					}
				}
			}
			if !allowed {
				http.NotFound(w, r)
				return
			}
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

		r = withTunnelEntryContext(r, entry)
		proxy.ServeHTTP(w, r)
	}
}

func applyHeaderTransforms(h http.Header, remove []string, add []headerKV) {
	for _, k := range remove {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		h.Del(k)
	}
	for _, kv := range add {
		k := strings.TrimSpace(kv.Name)
		if k == "" {
			continue
		}
		h.Set(k, kv.Value)
	}
}

func copyProxyForwardedHeaders(dst, src http.Header) {
	// ReverseProxy strips these before calling Rewrite; restore them from the
	// inbound request when proxy headers are trusted.
	for _, k := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
		if v, ok := src[k]; ok {
			dst[k] = append([]string(nil), v...)
		}
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

type tunnelEntryContextKey struct{}

func withTunnelEntryContext(r *http.Request, entry httpTunnelEntry) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), tunnelEntryContextKey{}, entry))
}

func tunnelEntryFromContext(ctx context.Context) (httpTunnelEntry, bool) {
	v := ctx.Value(tunnelEntryContextKey{})
	if v == nil {
		return httpTunnelEntry{}, false
	}
	entry, ok := v.(httpTunnelEntry)
	return entry, ok
}
