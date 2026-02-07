package client

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/inspect"
	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

type HeaderKV struct {
	Name  string
	Value string
}

type HTTPTunnelOptions struct {
	Authtoken            string
	Subdomain            string
	Domain               string
	BasicAuth            string
	AllowMethods         []string
	AllowPaths           []string
	AllowPathPrefixes    []string
	AllowCIDRs           []string
	DenyCIDRs            []string
	RequestHeaderAdd     []HeaderKV
	RequestHeaderRemove  []string
	ResponseHeaderAdd    []HeaderKV
	ResponseHeaderRemove []string
	HostHeader           string

	// UpstreamScheme is the scheme used when dialing the local upstream.
	// Supported values: "http" (default) and "https".
	UpstreamScheme string

	// UpstreamTLSSkipVerify disables certificate verification for HTTPS upstreams.
	// Ignored for non-HTTPS upstreams.
	UpstreamTLSSkipVerify bool

	Inspector *inspect.Store

	// CaptureBytes is the maximum number of bytes to keep for request and response
	// previews (used by the local inspector). If zero, a sensible default is used.
	CaptureBytes int
}

type HTTPTunnel struct {
	ID  string
	URL string

	localAddr            string
	controlURL           string
	authtoken            string
	subdomain            string
	domain               string
	basicAuth            string
	allowMethods         []string
	allowPaths           []string
	allowPathPrefixes    []string
	allowCIDRs           []string
	denyCIDRs            []string
	requestHeaderAdd     []HeaderKV
	requestHeaderRemove  []string
	responseHeaderAdd    []HeaderKV
	responseHeaderRemove []string
	hostHeader           string

	upstreamScheme        string
	upstreamTLSSkipVerify bool

	mu        sync.Mutex
	ws        *websocket.Conn
	session   *yamux.Session
	inspector *inspect.Store

	captureBytes int

	closing   atomic.Bool
	closeOnce sync.Once
	done      chan error
}

func StartHTTPTunnel(ctx context.Context, controlURL, localAddr string) (*HTTPTunnel, error) {
	return StartHTTPTunnelWithOptions(ctx, controlURL, localAddr, HTTPTunnelOptions{})
}

func StartHTTPTunnelWithOptions(ctx context.Context, controlURL, localAddr string, opts HTTPTunnelOptions) (*HTTPTunnel, error) {
	if err := ValidateHostHeaderMode(opts.HostHeader); err != nil {
		return nil, err
	}

	upstreamScheme := strings.ToLower(strings.TrimSpace(opts.UpstreamScheme))
	if upstreamScheme == "" {
		upstreamScheme = "http"
	}
	switch upstreamScheme {
	case "http", "https":
	default:
		return nil, errors.New("unsupported upstream scheme")
	}

	ws, session, resp, err := createHTTPTunnel(ctx, controlURL, control.CreateHTTPTunnelRequest{
		Type:                 "http",
		Authtoken:            opts.Authtoken,
		Subdomain:            opts.Subdomain,
		Domain:               opts.Domain,
		BasicAuth:            opts.BasicAuth,
		AllowMethod:          opts.AllowMethods,
		AllowPath:            opts.AllowPaths,
		AllowPathPrefix:      opts.AllowPathPrefixes,
		AllowCIDR:            opts.AllowCIDRs,
		DenyCIDR:             opts.DenyCIDRs,
		RequestHeaderAdd:     toControlHeaderKVs(opts.RequestHeaderAdd),
		RequestHeaderRemove:  append([]string(nil), opts.RequestHeaderRemove...),
		ResponseHeaderAdd:    toControlHeaderKVs(opts.ResponseHeaderAdd),
		ResponseHeaderRemove: append([]string(nil), opts.ResponseHeaderRemove...),
	})
	if err != nil {
		return nil, err
	}

	t := &HTTPTunnel{
		ID:                    resp.ID,
		URL:                   resp.URL,
		localAddr:             localAddr,
		controlURL:            controlURL,
		authtoken:             opts.Authtoken,
		subdomain:             opts.Subdomain,
		domain:                opts.Domain,
		basicAuth:             opts.BasicAuth,
		allowMethods:          append([]string(nil), opts.AllowMethods...),
		allowPaths:            append([]string(nil), opts.AllowPaths...),
		allowPathPrefixes:     append([]string(nil), opts.AllowPathPrefixes...),
		allowCIDRs:            append([]string(nil), opts.AllowCIDRs...),
		denyCIDRs:             append([]string(nil), opts.DenyCIDRs...),
		requestHeaderAdd:      append([]HeaderKV(nil), opts.RequestHeaderAdd...),
		requestHeaderRemove:   append([]string(nil), opts.RequestHeaderRemove...),
		responseHeaderAdd:     append([]HeaderKV(nil), opts.ResponseHeaderAdd...),
		responseHeaderRemove:  append([]string(nil), opts.ResponseHeaderRemove...),
		hostHeader:            opts.HostHeader,
		upstreamScheme:        upstreamScheme,
		upstreamTLSSkipVerify: opts.UpstreamTLSSkipVerify,
		ws:                    ws,
		session:               session,
		inspector:             opts.Inspector,
		captureBytes: func() int {
			if opts.CaptureBytes > 0 {
				return opts.CaptureBytes
			}
			return 64 * 1024
		}(),
		done: make(chan error, 1),
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

func (t *HTTPTunnel) Wait() error {
	return <-t.done
}

func (t *HTTPTunnel) acceptStreams(ctx context.Context) {
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

func (t *HTTPTunnel) finish(err error) {
	select {
	case t.done <- err:
	default:
	}
}

func (t *HTTPTunnel) handleStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()

	upstream, err := dialHTTPUpstream(ctx, t.upstreamScheme, t.localAddr, t.upstreamTLSSkipVerify)
	if err != nil {
		return
	}
	defer upstream.Close()

	hostHeader := strings.TrimSpace(t.hostHeader)
	if strings.EqualFold(hostHeader, "preserve") {
		hostHeader = ""
	}
	if strings.EqualFold(hostHeader, "rewrite") {
		hostHeader = t.localAddr
	}

	if t.inspector == nil {
		if hostHeader == "" {
			_ = proxyBidirectional(ctx, upstream, stream)
		} else {
			_, _, _ = proxyBidirectionalWithHostRewrite(ctx, upstream, stream, nil, nil, hostHeader)
		}
		return
	}

	startedAt := time.Now().UTC()

	reqCap := newPreviewCapture(t.captureBytes)
	respCap := newPreviewCapture(t.captureBytes)

	var bytesIn, bytesOut int64
	if hostHeader == "" {
		bytesIn, bytesOut, _ = proxyBidirectionalWithCapture(ctx, upstream, stream, reqCap, respCap)
	} else {
		bytesIn, bytesOut, _ = proxyBidirectionalWithHostRewrite(ctx, upstream, stream, reqCap, respCap, hostHeader)
	}
	duration := time.Since(startedAt)

	s, ok := summarizeHTTPExchange(reqCap.Bytes(), respCap.Bytes())
	if !ok {
		return
	}

	t.inspector.Add(inspect.Entry{
		StartedAt:       startedAt,
		DurationMs:      duration.Milliseconds(),
		TunnelID:        t.ID,
		Method:          s.Method,
		Path:            s.Path,
		Host:            s.Host,
		StatusCode:      s.StatusCode,
		BytesIn:         bytesIn,
		BytesOut:        bytesOut,
		RequestHeaders:  s.RequestHeaders,
		ResponseHeaders: s.ResponseHeaders,
	})
}

func dialHTTPUpstream(ctx context.Context, scheme, addr string, tlsSkipVerify bool) (net.Conn, error) {
	dialer := &net.Dialer{}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	if scheme != "https" {
		return conn, nil
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: tlsSkipVerify,
	})

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}

	return tlsConn, nil
}

func createHTTPTunnel(ctx context.Context, controlURL string, req control.CreateHTTPTunnelRequest) (*websocket.Conn, *yamux.Session, control.CreateHTTPTunnelResponse, error) {
	var resp control.CreateHTTPTunnelResponse

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

	resp, err = readJSONControlResponse[control.CreateHTTPTunnelResponse](ctrlStream)
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
	if resp.ID == "" || resp.URL == "" {
		_ = session.Close()
		_ = ws.Close(websocket.StatusInternalError, "invalid server response")
		return nil, nil, resp, errors.New("invalid server response")
	}

	return ws, session, resp, nil
}

func (t *HTTPTunnel) conn() (*websocket.Conn, *yamux.Session) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ws, t.session
}

func (t *HTTPTunnel) currentSession() *yamux.Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.session
}

func (t *HTTPTunnel) setConn(ws *websocket.Conn, session *yamux.Session) (*websocket.Conn, *yamux.Session) {
	t.mu.Lock()
	defer t.mu.Unlock()
	oldWS, oldSession := t.ws, t.session
	t.ws, t.session = ws, session
	return oldWS, oldSession
}

func (t *HTTPTunnel) controlRequestForReconnect() control.CreateHTTPTunnelRequest {
	req := control.CreateHTTPTunnelRequest{
		Type:                 "http",
		Authtoken:            t.authtoken,
		Subdomain:            t.subdomain,
		Domain:               t.domain,
		BasicAuth:            t.basicAuth,
		AllowMethod:          append([]string(nil), t.allowMethods...),
		AllowPath:            append([]string(nil), t.allowPaths...),
		AllowPathPrefix:      append([]string(nil), t.allowPathPrefixes...),
		AllowCIDR:            append([]string(nil), t.allowCIDRs...),
		DenyCIDR:             append([]string(nil), t.denyCIDRs...),
		RequestHeaderAdd:     toControlHeaderKVs(t.requestHeaderAdd),
		RequestHeaderRemove:  append([]string(nil), t.requestHeaderRemove...),
		ResponseHeaderAdd:    toControlHeaderKVs(t.responseHeaderAdd),
		ResponseHeaderRemove: append([]string(nil), t.responseHeaderRemove...),
	}

	if strings.TrimSpace(req.Domain) == "" && strings.TrimSpace(req.Subdomain) == "" {
		req.Domain = hostFromURL(t.URL)
	}

	return req
}

func toControlHeaderKVs(values []HeaderKV) []control.HeaderKV {
	if len(values) == 0 {
		return nil
	}

	out := make([]control.HeaderKV, 0, len(values))
	for _, kv := range values {
		out = append(out, control.HeaderKV{
			Name:  kv.Name,
			Value: kv.Value,
		})
	}
	return out
}

func (t *HTTPTunnel) reconnect(ctx context.Context) error {
	delay := 250 * time.Millisecond
	const maxDelay = 5 * time.Second

	for {
		if t.closing.Load() || ctx.Err() != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		}

		req := t.controlRequestForReconnect()

		ws, session, resp, err := createHTTPTunnel(ctx, t.controlURL, req)
		if err == nil {
			if t.closing.Load() || ctx.Err() != nil {
				_ = session.Close()
				_ = ws.Close(websocket.StatusNormalClosure, "closed")
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			}

			if resp.ID != t.ID || resp.URL != t.URL {
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

		// Retry only for likely-transient server-side errors.
		if isRetryableControlError(err) {
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

func isRetryableControlError(err error) bool {
	if err == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(err.Error())) {
	case "too many active tunnels", "rate limit exceeded":
		return true
	default:
		return false
	}
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		return u.Host
	}
	return raw
}
