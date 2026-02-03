package server

import (
	"net"
	"testing"
)

type fakeSession struct{}

func (fakeSession) OpenStream() (net.Conn, error) {
	a, b := net.Pipe()
	_ = b.Close()
	return a, nil
}

func (fakeSession) Close() error { return nil }

func TestTunnelRegistry_RegisterAndGetHTTP(t *testing.T) {
	t.Parallel()

	r := NewTunnelRegistry()

	if err := r.RegisterHTTPTunnel("abc123", fakeSession{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	sess, ok := r.GetHTTPTunnel("abc123")
	if !ok {
		t.Fatalf("expected tunnel to exist")
	}
	if sess == nil {
		t.Fatalf("expected session to be non-nil")
	}

	if err := r.RegisterHTTPTunnel("abc123", fakeSession{}); err == nil {
		t.Fatalf("expected duplicate id error")
	}

	r.UnregisterHTTPTunnel("abc123")
	_, ok = r.GetHTTPTunnel("abc123")
	if ok {
		t.Fatalf("expected tunnel to be removed")
	}
}

func TestTunnelRegistry_AllocateID(t *testing.T) {
	t.Parallel()

	r := NewTunnelRegistry()
	id, err := r.AllocateID()
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if len(id) != 8 {
		t.Fatalf("id len = %d, want %d", len(id), 8)
	}
}
