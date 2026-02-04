package server

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"net"
	"strings"
	"sync"
)

type TunnelRegistry struct {
	mu          sync.RWMutex
	httpTunnels map[string]httpTunnelEntry
}

type httpTunnelEntry struct {
	session   streamSession
	basicAuth *basicAuthCredential
}

type basicAuthCredential struct {
	Username string
	Password string
}

// streamSession is intentionally minimal and only supports opening a stream.
// We wrap yamux sessions to avoid importing yamux in other files.
type streamSession interface {
	OpenStream() (net.Conn, error)
	Close() error
}

func NewTunnelRegistry() *TunnelRegistry {
	return &TunnelRegistry{
		httpTunnels: make(map[string]httpTunnelEntry),
	}
}

func (r *TunnelRegistry) RegisterHTTPTunnel(id string, session streamSession, basicAuth *basicAuthCredential) error {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return errors.New("empty tunnel id")
	}
	if session == nil {
		return errors.New("nil session")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.httpTunnels[id]; exists {
		return errors.New("tunnel id already exists")
	}

	r.httpTunnels[id] = httpTunnelEntry{
		session:   session,
		basicAuth: basicAuth,
	}
	return nil
}

func (r *TunnelRegistry) GetHTTPTunnel(id string) (httpTunnelEntry, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return httpTunnelEntry{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.httpTunnels[id]
	if !ok {
		return httpTunnelEntry{}, false
	}
	return t, true
}

func (r *TunnelRegistry) UnregisterHTTPTunnel(id string) {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.httpTunnels, id)
}

func (r *TunnelRegistry) AllocateID() (string, error) {
	// 8 chars in base32 without padding gives a short, URL-safe id.
	const idLen = 8

	for i := 0; i < 10; i++ {
		id, err := randomBase32Lower(idLen)
		if err != nil {
			return "", err
		}

		r.mu.RLock()
		_, exists := r.httpTunnels[id]
		r.mu.RUnlock()

		if !exists {
			return id, nil
		}
	}

	return "", errors.New("failed to allocate unique id")
}

func randomBase32Lower(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid length")
	}

	// 5 bits per char. length chars needs ceil(length*5/8) bytes.
	nbytes := (length*5 + 7) / 8
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	s = strings.ToLower(s)

	// base32 may produce more chars than needed.
	if len(s) < length {
		return "", errors.New("internal error: short id")
	}
	return s[:length], nil
}
