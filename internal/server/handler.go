package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	BaseDomain   string
	TunnelDomain string
}

func ConfigFromEnv() Config {
	return Config{
		BaseDomain:   os.Getenv("EOSRIFT_BASE_DOMAIN"),
		TunnelDomain: os.Getenv("EOSRIFT_TUNNEL_DOMAIN"),
	}
}

func NewHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()

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
