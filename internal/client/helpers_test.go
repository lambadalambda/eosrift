package client

import (
	"errors"
	"testing"

	"eosrift.com/eosrift/internal/control"
)

func TestIsRetryableControlError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"too many tunnels", errors.New("too many active tunnels"), true},
		{"rate limit", errors.New("RATE LIMIT EXCEEDED"), true},
		{"other", errors.New("nope"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isRetryableControlError(tc.err); got != tc.want {
				t.Fatalf("got = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsRetryableTCPControlError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"requested port unavailable", errors.New("requested port unavailable"), true},
		{"too many tunnels", errors.New("too many active tunnels"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"other", errors.New("nope"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isRetryableTCPControlError(tc.err); got != tc.want {
				t.Fatalf("got = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHostFromURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"https://demo.tunnel.eosrift.com", "demo.tunnel.eosrift.com"},
		{"demo.tunnel.eosrift.com", "demo.tunnel.eosrift.com"},
		{"://bad", "://bad"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			if got := hostFromURL(tc.in); got != tc.want {
				t.Fatalf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestToControlHeaderKVs(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()

		if got := toControlHeaderKVs(nil); got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
	})

	t.Run("maps values", func(t *testing.T) {
		t.Parallel()

		got := toControlHeaderKVs([]HeaderKV{{Name: "X-Test", Value: "ok"}})
		want := []control.HeaderKV{{Name: "X-Test", Value: "ok"}}

		if len(got) != len(want) {
			t.Fatalf("len(got) = %d, want %d", len(got), len(want))
		}
		if got[0] != want[0] {
			t.Fatalf("got[0] = %#v, want %#v", got[0], want[0])
		}
	})
}

func TestHTTPTunnel_ControlRequestForReconnect_DefaultsDomain(t *testing.T) {
	t.Parallel()

	tun := &HTTPTunnel{
		URL:       "https://demo.tunnel.eosrift.com",
		authtoken: "tok",
	}

	req := tun.controlRequestForReconnect()
	if req.Type != "http" {
		t.Fatalf("type = %q, want %q", req.Type, "http")
	}
	if req.Domain != "demo.tunnel.eosrift.com" {
		t.Fatalf("domain = %q, want %q", req.Domain, "demo.tunnel.eosrift.com")
	}
}

func TestTCPTunnel_RemoteAddr(t *testing.T) {
	t.Parallel()

	tun := &TCPTunnel{RemotePort: 12345}
	got := tun.RemoteAddr("example.com")
	if got != "example.com:12345" {
		t.Fatalf("got = %q, want %q", got, "example.com:12345")
	}
}
