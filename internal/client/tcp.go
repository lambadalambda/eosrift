package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"eosrift.com/eosrift/internal/control"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type TCPTunnel struct {
	RemotePort int

	localAddr  string
	controlURL string
	authtoken  string

	mu      sync.Mutex
	ws      *websocket.Conn
	session *yamux.Session

	closing   atomic.Bool
	closeOnce sync.Once
	done      chan error
}

func StartTCPTunnel(ctx context.Context, controlURL, localAddr string) (*TCPTunnel, error) {
	return StartTCPTunnelWithOptions(ctx, controlURL, localAddr, TCPTunnelOptions{})
}

type TCPTunnelOptions struct {
	Authtoken  string
	RemotePort int
}

func StartTCPTunnelWithOptions(ctx context.Context, controlURL, localAddr string, opts TCPTunnelOptions) (*TCPTunnel, error) {
	ws, session, resp, err := createTCPTunnel(ctx, controlURL, control.CreateTCPTunnelRequest{
		Type:       "tcp",
		Authtoken:  opts.Authtoken,
		RemotePort: opts.RemotePort,
	})
	if err != nil {
		return nil, err
	}

	t := &TCPTunnel{
		RemotePort: resp.RemotePort,
		localAddr:  localAddr,
		controlURL: controlURL,
		authtoken:  opts.Authtoken,
		ws:         ws,
		session:    session,
		done:       make(chan error, 1),
	}

	go func() {
		<-ctx.Done()
		_ = t.Close()
	}()

	go t.acceptStreams(ctx)

	return t, nil
}

func (t *TCPTunnel) Close() error {
	var closeErr error

	t.closeOnce.Do(func() {
		t.closing.Store(true)
		ws, session := t.conn()
		if session != nil {
			closeErr = session.Close()
		}
		if ws != nil {
			_ = ws.Close(websocket.StatusNormalClosure, "closed")
		}
	})

	return closeErr
}

func (t *TCPTunnel) Wait() error {
	return <-t.done
}

func (t *TCPTunnel) acceptStreams(ctx context.Context) {
	for {
		session := t.currentSession()
		if session == nil {
			t.finish(errors.New("control session is nil"))
			return
		}

		stream, err := session.AcceptStream()
		if err != nil {
			if t.closing.Load() || ctx.Err() != nil {
				if ctx.Err() != nil {
					t.finish(ctx.Err())
				} else {
					t.finish(nil)
				}
				return
			}

			if err := t.reconnect(ctx); err != nil {
				t.finish(err)
				return
			}

			continue
		}

		go t.handleStream(ctx, stream)
	}
}

func (t *TCPTunnel) finish(err error) {
	select {
	case t.done <- err:
	default:
	}
}

func (t *TCPTunnel) handleStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()

	upstream, err := net.Dial("tcp", t.localAddr)
	if err != nil {
		return
	}
	defer upstream.Close()

	_ = proxyBidirectional(ctx, upstream, stream)
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

func (t *TCPTunnel) RemoteAddr(serverHost string) string {
	return fmt.Sprintf("%s:%d", serverHost, t.RemotePort)
}

func createTCPTunnel(ctx context.Context, controlURL string, req control.CreateTCPTunnelRequest) (*websocket.Conn, *yamux.Session, control.CreateTCPTunnelResponse, error) {
	var resp control.CreateTCPTunnelResponse

	ws, session, err := dialControlWithRetry(ctx, controlURL)
	if err != nil {
		return nil, nil, resp, err
	}

	ctrlStream, err := session.OpenStream()
	if err != nil {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, nil, resp, err
	}

	if err := control.WriteJSON(ctrlStream, req); err != nil {
		_ = ctrlStream.Close()
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, nil, resp, err
	}

	resp, err = readJSONControlResponse[control.CreateTCPTunnelResponse](ctrlStream)
	if err != nil {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, nil, resp, err
	}

	if resp.Error != "" {
		_ = session.Close()
		_ = ws.Close(websocket.StatusPolicyViolation, resp.Error)
		return nil, nil, resp, errors.New(resp.Error)
	}

	return ws, session, resp, nil
}

func (t *TCPTunnel) conn() (*websocket.Conn, *yamux.Session) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ws, t.session
}

func (t *TCPTunnel) currentSession() *yamux.Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.session
}

func (t *TCPTunnel) setConn(ws *websocket.Conn, session *yamux.Session) (*websocket.Conn, *yamux.Session) {
	t.mu.Lock()
	defer t.mu.Unlock()
	oldWS, oldSession := t.ws, t.session
	t.ws, t.session = ws, session
	return oldWS, oldSession
}

func (t *TCPTunnel) reconnect(ctx context.Context) error {
	delay := 250 * time.Millisecond
	const maxDelay = 5 * time.Second

	for {
		if t.closing.Load() || ctx.Err() != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		}

		ws, session, resp, err := createTCPTunnel(ctx, t.controlURL, control.CreateTCPTunnelRequest{
			Type:       "tcp",
			Authtoken:  t.authtoken,
			RemotePort: t.RemotePort,
		})
		if err == nil {
			if t.closing.Load() || ctx.Err() != nil {
				_ = session.Close()
				_ = ws.Close(websocket.StatusNormalClosure, "closed")
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			}

			if resp.RemotePort != t.RemotePort {
				_ = session.Close()
				_ = ws.Close(websocket.StatusInternalError, "resume mismatch")
				return errors.New("resume mismatch")
			}

			oldWS, oldSession := t.setConn(ws, session)
			if oldSession != nil {
				_ = oldSession.Close()
			}
			if oldWS != nil {
				_ = oldWS.Close(websocket.StatusGoingAway, "reconnected")
			}
			return nil
		}

		if isRetryableTCPControlError(err) {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		return err
	}
}

func isRetryableTCPControlError(err error) bool {
	if err == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(err.Error())) {
	case "requested port unavailable", "too many active tunnels", "rate limit exceeded":
		return true
	default:
		return false
	}
}
