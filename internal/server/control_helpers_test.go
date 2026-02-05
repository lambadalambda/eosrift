package server

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/control"
)

func TestParseBasicAuthCredential(t *testing.T) {
	t.Parallel()

	t.Run("empty is nil", func(t *testing.T) {
		t.Parallel()

		got, err := parseBasicAuthCredential("   ")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
	})

	t.Run("rejects missing colon", func(t *testing.T) {
		t.Parallel()

		if _, err := parseBasicAuthCredential("userpass"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects empty user", func(t *testing.T) {
		t.Parallel()

		if _, err := parseBasicAuthCredential(":pass"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("parses and trims user", func(t *testing.T) {
		t.Parallel()

		got, err := parseBasicAuthCredential(" user : pass ")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got == nil {
			t.Fatalf("got = nil, want non-nil")
		}
		if got.Username != "user" {
			t.Fatalf("username = %q, want %q", got.Username, "user")
		}
		// Password is not trimmed.
		if got.Password != " pass" {
			t.Fatalf("password = %q, want %q", got.Password, " pass")
		}
	})
}

func TestParseHeaderNameList(t *testing.T) {
	t.Parallel()

	got, err := parseHeaderNameList("request_header_remove", []string{"x-test"})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != "X-Test" {
		t.Fatalf("got = %#v, want %#v", got, []string{"X-Test"})
	}

	if _, err := parseHeaderNameList("request_header_remove", []string{"Host"}); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestParseHeaderKVList(t *testing.T) {
	t.Parallel()

	got, err := parseHeaderKVList("request_header_add", []control.HeaderKV{
		{Name: "x-test", Value: " ok "},
	})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Name != "X-Test" || got[0].Value != "ok" {
		t.Fatalf("got = %#v, want %#v", got, []headerKV{{Name: "X-Test", Value: "ok"}})
	}

	if _, err := parseHeaderKVList("request_header_add", []control.HeaderKV{{Name: "Host", Value: "example.com"}}); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestAllocateTCPListener(t *testing.T) {
	// Avoid t.Parallel: uses actual TCP ports.

	t.Run("rejects requested port out of range", func(t *testing.T) {
		cfg := Config{TCPPortRangeStart: 20000, TCPPortRangeEnd: 20010}
		if _, _, err := allocateTCPListener(cfg, 19999); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects invalid range for auto allocation", func(t *testing.T) {
		cfg := Config{TCPPortRangeStart: 0, TCPPortRangeEnd: 0}
		if _, _, err := allocateTCPListener(cfg, 0); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("requested port unavailable", func(t *testing.T) {
		occupied, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("listen occupied: %v", err)
		}
		t.Cleanup(func() { _ = occupied.Close() })

		_, portStr, err := net.SplitHostPort(occupied.Addr().String())
		if err != nil {
			t.Fatalf("split occupied addr: %v", err)
		}
		portStr = strings.TrimSpace(portStr)
		if portStr == "" {
			t.Fatalf("empty port")
		}

		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 {
			t.Fatalf("invalid occupied port %q", portStr)
		}

		cfg := Config{TCPPortRangeStart: port, TCPPortRangeEnd: port}
		if _, _, err := allocateTCPListener(cfg, port); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}
