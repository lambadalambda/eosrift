package cli

import (
	"net/http"
	"net/url"
)

func controlHost(controlURL string) string {
	u, err := url.Parse(controlURL)
	if err != nil {
		return controlURL
	}
	if h := u.Hostname(); h != "" {
		return h
	}
	return controlURL
}

func stripHopByHopHeaders(h http.Header) {
	// https://www.rfc-editor.org/rfc/rfc7230#section-6.1
	hopByHop := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, k := range hopByHop {
		h.Del(k)
	}
}
