package server

import (
	"net"
	"strings"
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

	if err := r.RegisterHTTPTunnel("abc123", fakeSession{}, httpTunnelOptions{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	entry, ok := r.GetHTTPTunnel("abc123")
	if !ok {
		t.Fatalf("expected tunnel to exist")
	}
	if entry.session == nil {
		t.Fatalf("expected session to be non-nil")
	}

	if err := r.RegisterHTTPTunnel("abc123", fakeSession{}, httpTunnelOptions{}); err == nil {
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

func TestTunnelRegistry_RegisterHTTPTunnel_RejectsEmptyID(t *testing.T) {
	t.Parallel()

	r := NewTunnelRegistry()
	if err := r.RegisterHTTPTunnel("", fakeSession{}, httpTunnelOptions{}); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestTunnelRegistry_RegisterHTTPTunnel_RejectsNilSession(t *testing.T) {
	t.Parallel()

	r := NewTunnelRegistry()
	if err := r.RegisterHTTPTunnel("abc123", nil, httpTunnelOptions{}); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestRandomBase32Lower(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid length", func(t *testing.T) {
		t.Parallel()

		if _, err := randomBase32Lower(0); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("returns lowercase base32", func(t *testing.T) {
		t.Parallel()

		id, err := randomBase32Lower(16)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(id) != 16 {
			t.Fatalf("len = %d, want %d", len(id), 16)
		}
		if strings.ToLower(id) != id {
			t.Fatalf("id not lowercase: %q", id)
		}
		for i := 0; i < len(id); i++ {
			c := id[i]
			if (c >= 'a' && c <= 'z') || (c >= '2' && c <= '7') {
				continue
			}
			t.Fatalf("unexpected character %q in id %q", c, id)
		}
	})
}
