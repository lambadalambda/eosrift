package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BaseDomain   string
	TunnelDomain string

	TCPPortRangeStart int
	TCPPortRangeEnd   int

	// DBPath is the path to the SQLite database.
	DBPath string

	// AuthToken, if set, is ensured to exist in the SQLite DB on startup.
	// (Bootstrap convenience; not required when tokens already exist.)
	AuthToken string
}

func ConfigFromEnv() Config {
	return Config{
		BaseDomain:   os.Getenv("EOSRIFT_BASE_DOMAIN"),
		TunnelDomain: os.Getenv("EOSRIFT_TUNNEL_DOMAIN"),

		TCPPortRangeStart: getenvInt("EOSRIFT_TCP_PORT_RANGE_START", 20000),
		TCPPortRangeEnd:   getenvInt("EOSRIFT_TCP_PORT_RANGE_END", 40000),

		DBPath: strings.TrimSpace(os.Getenv("EOSRIFT_DB_PATH")),

		AuthToken: strings.TrimSpace(os.Getenv("EOSRIFT_AUTH_TOKEN")),
	}
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type Dependencies struct {
	TokenValidator TokenValidator
}

func NewHandler(cfg Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	registry := NewTunnelRegistry()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/caddy/ask", func(w http.ResponseWriter, r *http.Request) {
		domain, err := caddyAskDomain(r)
		if err != nil {
			http.Error(w, "missing domain", http.StatusBadRequest)
			return
		}

		if cfg.BaseDomain == "" && cfg.TunnelDomain == "" {
			http.Error(w, "server misconfigured", http.StatusInternalServerError)
			return
		}

		if isAllowedDomain(domain, cfg) {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "forbidden", http.StatusForbidden)
	})

	mux.HandleFunc("/control", controlHandler(cfg, registry, deps.TokenValidator))
	mux.HandleFunc("/", httpTunnelProxyHandler(cfg, registry))

	return mux
}

func caddyAskDomain(r *http.Request) (string, error) {
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if domain == "" {
		domain = strings.TrimSpace(r.URL.Query().Get("host"))
	}
	if domain == "" {
		return "", errors.New("missing domain")
	}

	// Normalize.
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// Strip a port if one is present (best-effort).
	if strings.Contains(domain, ":") {
		if host, _, err := net.SplitHostPort(domain); err == nil {
			domain = host
		}
	}

	if domain == "" {
		return "", errors.New("empty domain")
	}

	return domain, nil
}

func isAllowedDomain(domain string, cfg Config) bool {
	base := strings.ToLower(strings.TrimSuffix(cfg.BaseDomain, "."))
	tunnel := strings.ToLower(strings.TrimSuffix(cfg.TunnelDomain, "."))

	switch {
	case base != "" && domain == base:
		return true
	case tunnel != "" && (domain == tunnel || strings.HasSuffix(domain, "."+tunnel)):
		return true
	default:
		return false
	}
}

// ValidateConfig returns an error if required configuration is missing or inconsistent.
func ValidateConfig(cfg Config) error {
	if cfg.BaseDomain == "" {
		return fmt.Errorf("EOSRIFT_BASE_DOMAIN is required")
	}
	if cfg.TunnelDomain == "" {
		return fmt.Errorf("EOSRIFT_TUNNEL_DOMAIN is required")
	}
	return nil
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
