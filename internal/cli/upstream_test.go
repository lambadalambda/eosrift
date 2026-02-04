package cli

import "testing"

func TestParseHTTPUpstreamTarget_Port(t *testing.T) {
	t.Parallel()

	scheme, addr, err := parseHTTPUpstreamTarget("3000")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if scheme != "http" {
		t.Fatalf("scheme = %q, want %q", scheme, "http")
	}
	if addr != "127.0.0.1:3000" {
		t.Fatalf("addr = %q, want %q", addr, "127.0.0.1:3000")
	}
}

func TestParseHTTPUpstreamTarget_HostPort(t *testing.T) {
	t.Parallel()

	scheme, addr, err := parseHTTPUpstreamTarget("localhost:3000")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if scheme != "http" {
		t.Fatalf("scheme = %q, want %q", scheme, "http")
	}
	if addr != "localhost:3000" {
		t.Fatalf("addr = %q, want %q", addr, "localhost:3000")
	}
}

func TestParseHTTPUpstreamTarget_URL_DefaultPort(t *testing.T) {
	t.Parallel()

	scheme, addr, err := parseHTTPUpstreamTarget("https://localhost")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if scheme != "https" {
		t.Fatalf("scheme = %q, want %q", scheme, "https")
	}
	if addr != "localhost:443" {
		t.Fatalf("addr = %q, want %q", addr, "localhost:443")
	}
}

func TestParseHTTPUpstreamTarget_URL_PathUnsupported(t *testing.T) {
	t.Parallel()

	_, _, err := parseHTTPUpstreamTarget("https://localhost:8443/foo")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTCPUpstreamAddr_Port(t *testing.T) {
	t.Parallel()

	addr, err := parseTCPUpstreamAddr("5432")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if addr != "127.0.0.1:5432" {
		t.Fatalf("addr = %q, want %q", addr, "127.0.0.1:5432")
	}
}

func TestParseTCPUpstreamAddr_URLRejected(t *testing.T) {
	t.Parallel()

	_, err := parseTCPUpstreamAddr("https://localhost:5432")
	if err == nil {
		t.Fatalf("expected error")
	}
}
