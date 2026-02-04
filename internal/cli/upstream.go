package cli

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func parseHTTPUpstreamTarget(input string) (scheme, addr string, _ error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", "", fmt.Errorf("empty upstream")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", "", fmt.Errorf("invalid upstream url: %w", err)
		}

		scheme = strings.ToLower(strings.TrimSpace(u.Scheme))
		switch scheme {
		case "http", "https":
		default:
			return "", "", fmt.Errorf("unsupported upstream scheme %q", u.Scheme)
		}

		if u.Hostname() == "" {
			return "", "", fmt.Errorf("invalid upstream url: missing host")
		}
		if u.User != nil {
			return "", "", fmt.Errorf("invalid upstream url: userinfo not supported")
		}
		if u.Path != "" && u.Path != "/" {
			return "", "", fmt.Errorf("invalid upstream url: path not supported")
		}
		if u.RawQuery != "" {
			return "", "", fmt.Errorf("invalid upstream url: query not supported")
		}
		if u.Fragment != "" {
			return "", "", fmt.Errorf("invalid upstream url: fragment not supported")
		}

		port := u.Port()
		if port == "" {
			if scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}

		if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
			return "", "", fmt.Errorf("invalid upstream url: invalid port")
		}

		return scheme, net.JoinHostPort(u.Hostname(), port), nil
	}

	if isPortLiteral(raw) {
		return "http", net.JoinHostPort("127.0.0.1", raw), nil
	}

	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid upstream addr %q: %v", input, err)
	}

	if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
		return "", "", fmt.Errorf("invalid upstream addr %q: invalid port", input)
	}

	return "http", net.JoinHostPort(host, port), nil
}

func parseTCPUpstreamAddr(input string) (string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", fmt.Errorf("empty upstream")
	}
	if strings.Contains(raw, "://") {
		return "", fmt.Errorf("invalid upstream addr %q: scheme not supported", input)
	}

	if isPortLiteral(raw) {
		return net.JoinHostPort("127.0.0.1", raw), nil
	}

	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("invalid upstream addr %q: %v", input, err)
	}

	if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
		return "", fmt.Errorf("invalid upstream addr %q: invalid port", input)
	}

	return net.JoinHostPort(host, port), nil
}

func isPortLiteral(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535
}
