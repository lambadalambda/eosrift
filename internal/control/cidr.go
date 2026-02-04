package control

import (
	"fmt"
	"net/netip"
	"strings"
)

func ParseCIDRList(field string, values []string, maxEntries int) ([]netip.Prefix, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if maxEntries > 0 && len(values) > maxEntries {
		return nil, fmt.Errorf("invalid %s: too many entries", field)
	}

	out := make([]netip.Prefix, 0, len(values))
	for _, v := range values {
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, fmt.Errorf("invalid %s", field)
		}

		var p netip.Prefix
		if strings.Contains(s, "/") {
			parsed, err := netip.ParsePrefix(s)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: %q", field, v)
			}
			p = parsed
		} else {
			a, err := netip.ParseAddr(s)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: %q", field, v)
			}
			a = a.Unmap()
			bits := 128
			if a.Is4() {
				bits = 32
			}
			p = netip.PrefixFrom(a, bits)
		}

		out = append(out, p.Masked())
	}

	return out, nil
}
