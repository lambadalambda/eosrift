package cli

import (
	"fmt"
	"net/netip"
	"strings"
)

func validateCIDRs(field string, values []string) error {
	for _, v := range values {
		s := strings.TrimSpace(v)
		if s == "" {
			return fmt.Errorf("invalid %s", field)
		}

		if strings.Contains(s, "/") {
			if _, err := netip.ParsePrefix(s); err != nil {
				return fmt.Errorf("invalid %s: %q", field, v)
			}
			continue
		}

		if _, err := netip.ParseAddr(s); err != nil {
			return fmt.Errorf("invalid %s: %q", field, v)
		}
	}

	return nil
}
