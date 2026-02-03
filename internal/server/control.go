package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"eosrift.com/eosrift/internal/control"
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
}

func controlHandler(cfg Config, registry *TunnelRegistry, deps Dependencies, limiter *tokenTunnelLimiter, rateLimiter *tokenRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		validator := deps.TokenValidator

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			log.Printf("control accept error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "closed")

		netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)

		session, err := yamux.Server(netConn, mux.QuietYamuxConfig())
		if err != nil {
			log.Printf("yamux server error: %v", err)
			return
		}
		defer session.Close()

		ctrlStream, err := session.AcceptStream()
		if err != nil {
			log.Printf("control accept stream error: %v", err)
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
			handleTCPControl(ctx, conn, session, ctrlStream, control.CreateTCPTunnelRequest{
				Type:       "tcp",
				Authtoken:  req.Authtoken,
				RemotePort: req.RemotePort,
			}, cfg)
			return
		case "http":
			handleHTTPControl(ctx, session, ctrlStream, control.CreateHTTPTunnelRequest{
				Type:      "http",
				Authtoken: req.Authtoken,
				Subdomain: req.Subdomain,
				Domain:    req.Domain,
			}, cfg, registry, deps, tokenID)
			return
		default:
			_ = writeControlTCPError(ctrlStream, "unsupported tunnel type")
			_ = ctrlStream.Close()
			return
		}
	}
}

func handleTCPControl(ctx context.Context, ws *websocket.Conn, session *yamux.Session, ctrlStream *yamux.Stream, req control.CreateTCPTunnelRequest, cfg Config) {
	ln, port, err := allocateTCPListener(cfg, req.RemotePort)
	if err != nil {
		_ = writeControlTCPError(ctrlStream, err.Error())
		_ = ctrlStream.Close()
		return
	}
	defer ln.Close()

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
		<-ctx.Done()
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
			log.Printf("tcp accept error: %v", err)
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

func handleHTTPControl(ctx context.Context, session *yamux.Session, ctrlStream *yamux.Stream, req control.CreateHTTPTunnelRequest, cfg Config, registry *TunnelRegistry, deps Dependencies, tokenID int64) {
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

	if err := registry.RegisterHTTPTunnel(id, yamuxSession{s: session}); err != nil {
		_ = json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelResponse{
			Type:  "http",
			Error: "failed to register tunnel",
		})
		_ = ctrlStream.Close()
		return
	}
	defer registry.UnregisterHTTPTunnel(id)

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

	<-ctx.Done()
	_ = session.Close()
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
