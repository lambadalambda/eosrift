package control

import (
	"fmt"
	"strings"
)

func ParseHTTPMethodList(field string, values []string, maxEntries int) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if maxEntries > 0 && len(values) > maxEntries {
		return nil, fmt.Errorf("invalid %s: too many entries", field)
	}

	out := make([]string, 0, len(values))
	for _, raw := range values {
		if strings.ContainsAny(raw, "\r\n\x00") {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}

		s := strings.TrimSpace(raw)
		if s == "" || !isValidHeaderToken(s) {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}

		out = append(out, strings.ToUpper(s))
	}

	return out, nil
}

func ParsePathList(field string, values []string, maxEntries int) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if maxEntries > 0 && len(values) > maxEntries {
		return nil, fmt.Errorf("invalid %s: too many entries", field)
	}

	out := make([]string, 0, len(values))
	for _, raw := range values {
		if strings.ContainsAny(raw, "\r\n\x00") {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}

		s := strings.TrimSpace(raw)
		if s == "" || !strings.HasPrefix(s, "/") {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}
		if strings.ContainsAny(s, "?#") {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}
		if !isSafePathValue(s) {
			return nil, fmt.Errorf("invalid %s: %q", field, raw)
		}

		out = append(out, s)
	}

	return out, nil
}

func isSafePathValue(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' || c == '\n' || c == 0 {
			return false
		}
		// Disallow ASCII control characters and whitespace.
		if c <= 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

