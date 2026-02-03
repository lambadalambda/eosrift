package inspect

import (
	"net/http"
	"net/url"
	"strings"
)

const RedactedValue = "REDACTED"

var sensitiveHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"Set-Cookie",
	"X-Api-Key",
	"X-Auth-Token",
	"X-Access-Token",
}

func redactEntry(e Entry) Entry {
	e.Path = redactPath(e.Path)
	e.RequestHeaders = redactHeaders(e.RequestHeaders)
	e.ResponseHeaders = redactHeaders(e.ResponseHeaders)
	return e
}

func redactHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}

	out := h.Clone()
	for _, name := range sensitiveHeaders {
		if out.Get(name) != "" {
			out.Set(name, RedactedValue)
		}
	}
	return out
}

func redactPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	u, err := url.ParseRequestURI(path)
	if err != nil {
		return path
	}

	q := u.Query()
	changed := false
	for key, vals := range q {
		if !isSensitiveQueryKey(key) {
			continue
		}
		changed = true
		for i := range vals {
			vals[i] = RedactedValue
		}
		q[key] = vals
	}

	if changed {
		u.RawQuery = q.Encode()
	}
	return u.RequestURI()
}

func isSensitiveQueryKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case "token", "access_token", "refresh_token", "auth", "apikey", "api_key", "key", "signature", "sig", "password", "pass", "passwd", "secret":
		return true
	default:
		return false
	}
}
