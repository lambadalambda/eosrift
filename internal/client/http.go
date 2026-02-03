package client

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"

	"eosrift.com/eosrift/internal/control"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type HTTPTunnel struct {
	ID  string
	URL string

	localAddr string
	ws        *websocket.Conn
	session   *yamux.Session

	closeOnce sync.Once
	done      chan error
}

func StartHTTPTunnel(ctx context.Context, controlURL, localAddr string) (*HTTPTunnel, error) {
	ws, session, err := dialControl(ctx, controlURL)
	if err != nil {
		return nil, err
	}

	ctrlStream, err := session.OpenStream()
	if err != nil {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, err
	}

	if err := json.NewEncoder(ctrlStream).Encode(control.CreateHTTPTunnelRequest{
		Type: "http",
	}); err != nil {
		_ = ctrlStream.Close()
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return nil, err
	}

	var resp control.CreateHTTPTunnelResponse
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
	if resp.ID == "" || resp.URL == "" {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "invalid server response")
		return nil, errors.New("invalid server response")
	}

	t := &HTTPTunnel{
		ID:        resp.ID,
		URL:       resp.URL,
		localAddr: localAddr,
		ws:        ws,
		session:   session,
		done:      make(chan error, 1),
	}

	go func() {
		<-ctx.Done()
		_ = t.Close()
	}()

	go t.acceptStreams(ctx)

	return t, nil
}

func (t *HTTPTunnel) Close() error {
	var closeErr error

	t.closeOnce.Do(func() {
		if t.session != nil {
			closeErr = t.session.Close()
		}
		if t.ws != nil {
			_ = t.ws.Close(websocket.StatusNormalClosure, "closed")
		}
	})

	return closeErr
}

func (t *HTTPTunnel) Wait() error {
	return <-t.done
}

func (t *HTTPTunnel) acceptStreams(ctx context.Context) {
	defer func() {
		select {
		case t.done <- ctx.Err():
		default:
		}
	}()

	for {
		stream, err := t.session.AcceptStream()
		if err != nil {
			select {
			case t.done <- err:
			default:
			}
			return
		}

		go t.handleStream(ctx, stream)
	}
}

func (t *HTTPTunnel) handleStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()

	upstream, err := net.Dial("tcp", t.localAddr)
	if err != nil {
		return
	}
	defer upstream.Close()

	_ = proxyBidirectional(ctx, upstream, stream)
}

