package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"eosrift.com/eosrift/internal/control"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type TCPTunnel struct {
	RemotePort int

	localAddr string
	ws        *websocket.Conn
	session   *yamux.Session

	closing   atomic.Bool
	closeOnce sync.Once
	done      chan error
}

func StartTCPTunnel(ctx context.Context, controlURL, localAddr string) (*TCPTunnel, error) {
	return StartTCPTunnelWithOptions(ctx, controlURL, localAddr, TCPTunnelOptions{})
}

type TCPTunnelOptions struct {
	Authtoken string
}

func StartTCPTunnelWithOptions(ctx context.Context, controlURL, localAddr string, opts TCPTunnelOptions) (*TCPTunnel, error) {
	ws, session, err := dialControlWithRetry(ctx, controlURL)
	if err != nil {
		return nil, err
	}

	ctrlStream, err := session.OpenStream()
	if err != nil {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, err
	}

	if err := json.NewEncoder(ctrlStream).Encode(control.CreateTCPTunnelRequest{
		Type:       "tcp",
		Authtoken:  opts.Authtoken,
		RemotePort: 0,
	}); err != nil {
		_ = ctrlStream.Close()
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, err
	}

	var resp control.CreateTCPTunnelResponse
	if err := json.NewDecoder(ctrlStream).Decode(&resp); err != nil {
		_ = ctrlStream.Close()
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, err
	}
	_ = ctrlStream.Close()

	if resp.Error != "" {
		_ = session.Close()
		_ = ws.Close(websocket.StatusPolicyViolation, resp.Error)
		return nil, errors.New(resp.Error)
	}

	t := &TCPTunnel{
		RemotePort: resp.RemotePort,
		localAddr:  localAddr,
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
		if t.session != nil {
			closeErr = t.session.Close()
		}
		if t.ws != nil {
			_ = t.ws.Close(websocket.StatusNormalClosure, "closed")
		}
	})

	return closeErr
}

func (t *TCPTunnel) Wait() error {
	return <-t.done
}

func (t *TCPTunnel) acceptStreams(ctx context.Context) {
	for {
		stream, err := t.session.AcceptStream()
		if err != nil {
			if t.closing.Load() || ctx.Err() != nil {
				if ctx.Err() != nil {
					t.finish(ctx.Err())
				} else {
					t.finish(nil)
				}
				return
			}
			t.finish(err)
			return
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
