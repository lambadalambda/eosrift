package control

import (
	"fmt"
	"net/http"
	"strings"
)

const MaxHeaderValueBytes = 8 * 1024

func NormalizeHeaderName(field string, raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" || !isValidHeaderToken(s) {
		return "", fmt.Errorf("invalid %s: %q", field, raw)
	}

	s = http.CanonicalHeaderKey(s)
	if isDisallowedTransformedHeader(s) {
		return "", fmt.Errorf("invalid %s: %q", field, raw)
	}

	return s, nil
}

func ValidateHeaderValue(field, raw string, value string) (string, error) {
	v := strings.TrimSpace(value)
	if len(v) > MaxHeaderValueBytes || !isSafeHeaderValue(v) {
		return "", fmt.Errorf("invalid %s: %q", field, raw)
	}
	return v, nil
}

func isValidHeaderToken(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)):
		default:
			return false
		}
	}
	return true
}

func isSafeHeaderValue(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' || c == '\n' || c == 0 {
			return false
		}
		if c < 0x20 && c != '\t' {
			return false
		}
		if c == 0x7f {
			return false
		}
	}
	return true
}

func isDisallowedTransformedHeader(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
		"Content-Length",
		"Host":
		return true
	default:
		return false
	}
}
