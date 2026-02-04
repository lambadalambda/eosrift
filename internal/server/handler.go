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
	"time"

	"eosrift.com/eosrift/internal/logging"
)

type Config struct {
	BaseDomain   string
	TunnelDomain string

	TCPPortRangeStart int
	TCPPortRangeEnd   int

	// MetricsToken enables /metrics when set (requires Authorization: Bearer <token>).
	MetricsToken string

	// MaxTunnelsPerToken caps concurrent tunnels per authtoken.
	// Zero means unlimited.
	MaxTunnelsPerToken int

	// MaxTunnelCreatesPerMinute caps tunnel create attempts per authtoken.
	// Zero means unlimited.
	MaxTunnelCreatesPerMinute int

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

		MetricsToken: strings.TrimSpace(os.Getenv("EOSRIFT_METRICS_TOKEN")),

		MaxTunnelsPerToken: getenvInt("EOSRIFT_MAX_TUNNELS_PER_TOKEN", 0),

		MaxTunnelCreatesPerMinute: getenvInt("EOSRIFT_MAX_TUNNEL_CREATES_PER_MIN", 0),

		DBPath: strings.TrimSpace(os.Getenv("EOSRIFT_DB_PATH")),

		AuthToken: strings.TrimSpace(os.Getenv("EOSRIFT_AUTH_TOKEN")),
	}
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type TokenResolver interface {
	TokenID(ctx context.Context, token string) (int64, bool, error)
}

type ReservationStore interface {
	ReservedSubdomainTokenID(ctx context.Context, subdomain string) (int64, bool, error)
	ReserveSubdomain(ctx context.Context, tokenID int64, subdomain string) error
}

type Dependencies struct {
	TokenValidator TokenValidator
	TokenResolver  TokenResolver
	Reservations   ReservationStore
	Logger         logging.Logger
}

func NewHandler(cfg Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	registry := NewTunnelRegistry()
	tunnelProxy := httpTunnelProxyHandler(cfg, registry)
	limiter := newTokenTunnelLimiter()
	rateLimiter := newTokenRateLimiter(time.Now)
	metrics := newMetrics(time.Now)

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

		base := normalizeDomain(cfg.BaseDomain)
		tunnel := normalizeDomain(cfg.TunnelDomain)

		// Always allow issuing a cert for the base domain and tunnel domain apex.
		if base != "" && domain == base {
			w.WriteHeader(http.StatusOK)
			return
		}
		if tunnel != "" && domain == tunnel {
			w.WriteHeader(http.StatusOK)
			return
		}

		// For tunnel subdomains, only allow issuance if:
		// - the tunnel is currently active (in-memory), or
		// - the subdomain is reserved in SQLite.
		//
		// This prevents arbitrary third parties from forcing ACME issuance for random
		// hostnames under the tunnel domain.
		if id, ok := tunnelIDFromHost(domain, cfg.TunnelDomain); ok {
			if _, ok := registry.GetHTTPTunnel(id); ok {
				w.WriteHeader(http.StatusOK)
				return
			}
			if deps.Reservations != nil {
				if _, reserved, err := deps.Reservations.ReservedSubdomainTokenID(r.Context(), id); err == nil && reserved {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		http.Error(w, "forbidden", http.StatusForbidden)
	})

	if cfg.MetricsToken != "" {
		mux.HandleFunc("/metrics", metricsHandler(cfg.BaseDomain, cfg.MetricsToken, metrics))
	}

	mux.HandleFunc("/control", controlHandler(cfg, registry, deps, limiter, rateLimiter, metrics))
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		if isBaseDomainHost(r.Host, cfg.BaseDomain) && r.URL.Path == "/style.css" {
			serveLandingStyle(w, r)
			return
		}
		tunnelProxy(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if isBaseDomainHost(r.Host, cfg.BaseDomain) && r.URL.Path == "/" {
			serveLandingIndex(w, r)
			return
		}
		tunnelProxy(w, r)
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

func isBaseDomainHost(host, baseDomain string) bool {
	base := normalizeDomain(baseDomain)
	if base == "" {
		return false
	}
	return normalizeDomain(host) == base
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
