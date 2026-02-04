//go:build integration

package integration

import (
	"net/url"
	"os"
	"strings"
)

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func controlURL() string {
	return getenv("EOSRIFT_CONTROL_URL", "ws://server:8080/control")
}

func httpBaseURL() string {
	base := getenv("EOSRIFT_HTTP_BASE_URL", "")
	if base == "" {
		base = getenv("EOSRIFT_SERVER_URL", "http://server:8080")
	}
	return strings.TrimRight(base, "/")
}

func httpURL(path string) string {
	base := httpBaseURL()
	u, err := url.Parse(base)
	if err != nil {
		return base + path
	}

	basePath := strings.TrimRight(u.Path, "/")
	if path == "" {
		u.Path = basePath
		return u.String()
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = basePath + path
	return u.String()
}

func wsURL(path string) string {
	base := httpBaseURL()
	u, err := url.Parse(base)
	if err != nil {
		return base + path
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	basePath := strings.TrimRight(u.Path, "/")
	if path == "" {
		u.Path = basePath
		return u.String()
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = basePath + path
	return u.String()
}

func tcpDialHost() string {
	return getenv("EOSRIFT_TCP_DIAL_HOST", "server")
}

func testNetworkCIDR() string {
	// Kept as an env var so our integration tests can run in multiple Compose
	// networks (e.g. with and without Caddy in front).
	return getenv("EOSRIFT_TEST_CIDR", "10.231.0.0/24")
}
