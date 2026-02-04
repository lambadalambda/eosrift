package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/logging"
	"eosrift.com/eosrift/internal/mux"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type yamuxSession struct {
	s *yamux.Session
}

func (y yamuxSession) OpenStream() (net.Conn, error) {
	st, err := y.s.OpenStream()
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (y yamuxSession) Close() error {
	return y.s.Close()
}

type baseRequest struct {
	Type       string `json:"type"`
	Authtoken  string `json:"authtoken,omitempty"`
	RemotePort int    `json:"remote_port,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	Domain     string `json:"domain,omitempty"`
	BasicAuth  string `json:"basic_auth,omitempty"`

	AllowCIDR []string `json:"allow_cidr,omitempty"`
	DenyCIDR  []string `json:"deny_cidr,omitempty"`
}

func controlHandler(cfg Config, registry *TunnelRegistry, deps Dependencies, limiter *tokenTunnelLimiter, rateLimiter *tokenRateLimiter, metrics *metrics) http.HandlerFunc {
	logger := deps.Logger
	if logger == nil {
		logger = logging.New(logging.Options{})
	}
	logger = logger.With(logging.F("component", "control"))

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		validator := deps.TokenValidator

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			logger.Warn("control accept error", logging.F("err", err), logging.F("remote_addr", r.RemoteAddr))
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "closed")

		reqLogger := logger.With(logging.F("remote_addr", r.RemoteAddr))

		var releaseControlConn func()
		if metrics != nil {
			releaseControlConn = metrics.trackControlConn()
			defer releaseControlConn()
		}

		netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)

		session, err := yamux.Server(netConn, mux.QuietYamuxConfig())
		if err != nil {
			reqLogger.Warn("yamux server error", logging.F("err", err))
			return
		}
		defer session.Close()

		ctrlStream, err := session.AcceptStream()
		if err != nil {
			reqLogger.Warn("control accept stream error", logging.F("err", err))
			return
		}

		var req baseRequest
		if err := json.NewDecoder(ctrlStream).Decode(&req); err != nil {
			_ = writeControlTCPError(ctrlStream, "invalid request")
			_ = ctrlStream.Close()
			return
		}

		reqType := strings.ToLower(strings.TrimSpace(req.Type))
		if validator != nil {
			ok, err := validator.ValidateToken(ctx, req.Authtoken)
			if err != nil {
				switch reqType {
				case "tcp":
					_ = writeControlTCPError(ctrlStream, "auth error")
				case "http":
					_ = writeControlHTTPError(ctrlStream, "auth error")
				default:
					_ = writeControlTCPError(ctrlStream, "auth error")
				}
				_ = ctrlStream.Close()
				return
			}
			if !ok {
				switch reqType {
				case "tcp":
					_ = writeControlTCPError(ctrlStream, "unauthorized")
				case "http":
					_ = writeControlHTTPError(ctrlStream, "unauthorized")
				default:
					_ = writeControlTCPError(ctrlStream, "unauthorized")
				}
				_ = ctrlStream.Close()
				return
			}
		}

		if validator == nil && cfg.AuthToken != "" && strings.TrimSpace(req.Authtoken) != cfg.AuthToken {
			switch reqType {
			case "tcp":
				_ = writeControlTCPError(ctrlStream, "unauthorized")
			case "http":
				_ = writeControlHTTPError(ctrlStream, "unauthorized")
			default:
				_ = writeControlTCPError(ctrlStream, "unauthorized")
			}
			_ = ctrlStream.Close()
			return
		}

		var tokenID int64
		if deps.TokenResolver != nil {
			id, ok, err := deps.TokenResolver.TokenID(ctx, req.Authtoken)
			if err != nil {
				switch reqType {
				case "tcp":
					_ = writeControlTCPError(ctrlStream, "auth error")
				case "http":
					_ = writeControlHTTPError(ctrlStream, "auth error")
				default:
					_ = writeControlTCPError(ctrlStream, "auth error")
				}
				_ = ctrlStream.Close()
				return
			}
			if ok {
				tokenID = id
			}
		}

		if cfg.MaxTunnelsPerToken > 0 && limiter != nil && tokenID > 0 {
			release, ok := limiter.TryAcquire(tokenID, cfg.MaxTunnelsPerToken)
			if !ok {
				switch reqType {
				case "tcp":
					_ = writeControlTCPError(ctrlStream, "too many active tunnels")
				case "http":
					_ = writeControlHTTPError(ctrlStream, "too many active tunnels")
				default:
					_ = writeControlTCPError(ctrlStream, "too many active tunnels")
				}
				_ = ctrlStream.Close()
				return
			}
			defer release()
		}

		if cfg.MaxTunnelCreatesPerMinute > 0 && rateLimiter != nil && tokenID > 0 {
			if !rateLimiter.Allow(tokenID, cfg.MaxTunnelCreatesPerMinute) {
				switch reqType {
				case "tcp":
					_ = writeControlTCPError(ctrlStream, "rate limit exceeded")
				case "http":
					_ = writeControlHTTPError(ctrlStream, "rate limit exceeded")
				default:
					_ = writeControlTCPError(ctrlStream, "rate limit exceeded")
				}
				_ = ctrlStream.Close()
				return
			}
		}

		switch reqType {
		case "tcp":
			if req.RemotePort != 0 && tokenID > 0 && deps.Reservations != nil {
				if req.RemotePort < cfg.TCPPortRangeStart || req.RemotePort > cfg.TCPPortRangeEnd {
					_ = writeControlTCPError(ctrlStream, "requested port out of range")
					_ = ctrlStream.Close()
					return
				}

				reservedTokenID, reserved, err := deps.Reservations.ReservedTCPPortTokenID(ctx, req.RemotePort)
				if err != nil {
					_ = writeControlTCPError(ctrlStream, "invalid requested port")
					_ = ctrlStream.Close()
					return
				}
				if reserved && reservedTokenID != tokenID {
					_ = writeControlTCPError(ctrlStream, "unauthorized")
					_ = ctrlStream.Close()
					return
				}

				if !reserved {
					if err := deps.Reservations.ReserveTCPPort(ctx, tokenID, req.RemotePort); err != nil {
						// In case of a race, re-check ownership.
						reservedTokenID, reserved, err2 := deps.Reservations.ReservedTCPPortTokenID(ctx, req.RemotePort)
						if err2 == nil && reserved && reservedTokenID == tokenID {
							// OK: claimed by us.
						} else if err2 == nil && reserved && reservedTokenID != tokenID {
							_ = writeControlTCPError(ctrlStream, "unauthorized")
							_ = ctrlStream.Close()
							return
						} else {
							_ = writeControlTCPError(ctrlStream, "failed to reserve port")
							_ = ctrlStream.Close()
							return
						}
					}
				}
			}

			handleTCPControl(ctx, conn, session, ctrlStream, control.CreateTCPTunnelRequest{
				Type:       "tcp",
				Authtoken:  req.Authtoken,
				RemotePort: req.RemotePort,
			}, cfg, metrics, reqLogger)
			return
		case "http":
			handleHTTPControl(ctx, session, ctrlStream, control.CreateHTTPTunnelRequest{
				Type:      "http",
				Authtoken: req.Authtoken,
				Subdomain: req.Subdomain,
				Domain:    req.Domain,
				BasicAuth: req.BasicAuth,
				AllowCIDR: req.AllowCIDR,
				DenyCIDR:  req.DenyCIDR,
			}, cfg, registry, deps, tokenID, metrics)
			return
		default:
			_ = writeControlTCPError(ctrlStream, "unsupported tunnel type")
			_ = ctrlStream.Close()
			return
		}
	}
}

func handleTCPControl(ctx context.Context, ws *websocket.Conn, session *yamux.Session, ctrlStream *yamux.Stream, req control.CreateTCPTunnelRequest, cfg Config, metrics *metrics, logger logging.Logger) {
	ln, port, err := allocateTCPListener(cfg, req.RemotePort)
	if err != nil {
		_ = writeControlTCPError(ctrlStream, err.Error())
		_ = ctrlStream.Close()
		return
	}
	defer ln.Close()

	var releaseTunnel func()
	if metrics != nil {
		releaseTunnel = metrics.trackTCPTunnel()
		defer releaseTunnel()
	}

	if err := json.NewEncoder(ctrlStream).Encode(control.CreateTCPTunnelResponse{
		Type:       "tcp",
		RemotePort: port,
	}); err != nil {
		_ = ctrlStream.Close()
		return
	}
	_ = ctrlStream.Close()

	// Ensure listener is closed on websocket disconnect.
	go func() {
		select {
		case <-ctx.Done():
		case <-session.CloseChan():
		}
		_ = ln.Close()
		_ = session.Close()
	}()

	for {
		inbound, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				_ = ws.Close(websocket.StatusNormalClosure, "closed")
				return
			}
			if logger != nil {
				logger.Warn("tcp accept error", logging.F("err", err))
			}
			return
		}

		go func(in net.Conn) {
			defer in.Close()

			stream, err := session.OpenStream()
			if err != nil {
				return
			}
			defer stream.Close()

			_ = proxyBidirectional(ctx, in, stream)
		}(inbound)
	}
}

func handleHTTPControl(ctx context.Context, session *yamux.Session, ctrlStream *yamux.Stream, req control.CreateHTTPTunnelRequest, cfg Config, registry *TunnelRegistry, deps Dependencies, tokenID int64, metrics *metrics) {
	id, err := func() (string, error) {
		domain := strings.TrimSpace(req.Domain)
		subdomain := strings.TrimSpace(req.Subdomain)

		switch {
		case domain == "" && subdomain == "":
			id, err := registry.AllocateID()
			if err != nil {
				return "", errors.New("failed to allocate id")
			}
			return id, nil
		case domain != "" && subdomain != "":
			return "", errors.New("invalid request")
		}

		if tokenID <= 0 || deps.Reservations == nil {
			return "", errors.New("unauthorized")
		}

		desired := subdomain
		if domain != "" {
			host := domain
			if strings.Contains(host, "://") {
				u, err := url.Parse(host)
				if err != nil {
					return "", errors.New("invalid domain")
				}
				host = u.Host
			}

			id, ok := tunnelIDFromHost(host, cfg.TunnelDomain)
			if !ok {
				return "", errors.New("invalid domain")
			}
			desired = id
		}

		reservedTokenID, reserved, err := deps.Reservations.ReservedSubdomainTokenID(ctx, desired)
		if err != nil {
			return "", errors.New("invalid subdomain")
		}
		if reserved && reservedTokenID != tokenID {
			return "", errors.New("unauthorized")
		}

		if !reserved {
			if err := deps.Reservations.ReserveSubdomain(ctx, tokenID, desired); err != nil {
				// In case of a race, re-check ownership.
				reservedTokenID, reserved, err2 := deps.Reservations.ReservedSubdomainTokenID(ctx, desired)
				if err2 == nil && reserved && reservedTokenID == tokenID {
					return desired, nil
				}
				if err2 == nil && reserved && reservedTokenID != tokenID {
					return "", errors.New("unauthorized")
				}
				return "", errors.New("failed to reserve subdomain")
			}
		}

		return desired, nil
	}()
	if err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: err.Error(),
		})
		_ = ctrlStream.Close()
		return
	}

	basicAuth, err := parseBasicAuthCredential(req.BasicAuth)
	if err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: err.Error(),
		})
		_ = ctrlStream.Close()
		return
	}

	allowCIDRs, err := parseCIDRList("allow_cidr", req.AllowCIDR)
	if err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: err.Error(),
		})
		_ = ctrlStream.Close()
		return
	}
	denyCIDRs, err := parseCIDRList("deny_cidr", req.DenyCIDR)
	if err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: err.Error(),
		})
		_ = ctrlStream.Close()
		return
	}

	if err := registry.RegisterHTTPTunnel(id, yamuxSession{s: session}, basicAuth, allowCIDRs, denyCIDRs); err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: "failed to register tunnel",
		})
		_ = ctrlStream.Close()
		return
	}
	defer registry.UnregisterHTTPTunnel(id)

	var releaseTunnel func()
	if metrics != nil {
		releaseTunnel = metrics.trackHTTPTunnel()
		defer releaseTunnel()
	}

	url := fmt.Sprintf("https://%s.%s", id, strings.TrimSuffix(cfg.TunnelDomain, "."))
	if err := json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
		Type: "http",
		ID:   id,
		URL:  url,
	}); err != nil {
		_ = ctrlStream.Close()
		return
	}
	_ = ctrlStream.Close()

	select {
	case <-ctx.Done():
	case <-session.CloseChan():
	}
	_ = session.Close()
}

func parseBasicAuthCredential(s string) (*basicAuthCredential, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	user, pass, ok := strings.Cut(s, ":")
	if !ok {
		return nil, errors.New("invalid basic_auth")
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return nil, errors.New("invalid basic_auth")
	}

	return &basicAuthCredential{Username: user, Password: pass}, nil
}

func parseCIDRList(field string, values []string) ([]netip.Prefix, error) {
	if len(values) == 0 {
		return nil, nil
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

func writeControlTCPError(w io.Writer, msg string) error {
	enc := json.NewEncoder(w)
	return enc.Encode(control.CreateTCPTunnelResponse{
		Type:  "tcp",
		Error: msg,
	})
}

func writeControlHTTPError(w io.Writer, msg string) error {
	enc := json.NewEncoder(w)
	return enc.Encode(control.CreateHTTPTunnelResponse{
		Type:  "http",
		Error: msg,
	})
}

func allocateTCPListener(cfg Config, requestedPort int) (net.Listener, int, error) {
	if requestedPort != 0 {
		if requestedPort < cfg.TCPPortRangeStart || requestedPort > cfg.TCPPortRangeEnd {
			return nil, 0, fmt.Errorf("requested port out of range")
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", requestedPort))
		if err != nil {
			return nil, 0, fmt.Errorf("requested port unavailable")
		}
		return ln, requestedPort, nil
	}

	start := cfg.TCPPortRangeStart
	end := cfg.TCPPortRangeEnd
	if start <= 0 || end <= 0 || end < start {
		return nil, 0, fmt.Errorf("invalid tcp port range")
	}

	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		return ln, port, nil
	}

	return nil, 0, fmt.Errorf("no ports available")
}

func proxyBidirectional(ctx context.Context, a, b net.Conn) error {
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(a, b)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(b, a)
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		_ = a.SetDeadline(time.Now())
		_ = b.SetDeadline(time.Now())
		return ctx.Err()
	case err := <-errCh:
		_ = a.SetDeadline(time.Now())
		_ = b.SetDeadline(time.Now())
		return err
	}
}
