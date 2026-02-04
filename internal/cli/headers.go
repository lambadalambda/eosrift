package cli

import (
	"fmt"
	"net/http"
	"strings"

	"eosrift.com/eosrift/internal/client"
)

func parseHeaderAddList(field string, values []string) ([]client.HeaderKV, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]client.HeaderKV, 0, len(values))
	for _, raw := range values {
		kv, err := parseHeaderKV(field, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, kv)
	}
	return out, nil
}

func parseHeaderRemoveList(field string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(values))
	for _, raw := range values {
		name, err := normalizeHeaderName(field, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, nil
}

func parseHeaderKV(field string, raw string) (client.HeaderKV, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return client.HeaderKV{}, fmt.Errorf("invalid %s: %q", field, raw)
	}

	name, value, ok := strings.Cut(s, ":")
	if !ok {
		name, value, ok = strings.Cut(s, "=")
	}
	if !ok {
		return client.HeaderKV{}, fmt.Errorf("invalid %s: %q", field, raw)
	}

	normName, err := normalizeHeaderName(field, name)
	if err != nil {
		return client.HeaderKV{}, err
	}

	return client.HeaderKV{
		Name:  normName,
		Value: strings.TrimSpace(value),
	}, nil
}

func normalizeHeaderName(field string, raw string) (string, error) {
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
