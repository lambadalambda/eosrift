package cli

import (
	"net"
	"testing"
)

func TestListenTCPWithPortFallback_BumpsPortOnInUse(t *testing.T) {
	t.Parallel()

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen occupied: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	_, port, err := net.SplitHostPort(occupied.Addr().String())
	if err != nil {
		t.Fatalf("split occupied addr: %v", err)
	}

	addr := "127.0.0.1:" + port
	ln, err := listenTCPWithPortFallback(addr, 0)
	if err == nil {
		_ = ln.Close()
		t.Fatalf("expected error due to maxPort==startPort, got listener")
	}

	ln, err = listenTCPWithPortFallback(addr, 65535)
	if err != nil {
		t.Fatalf("listen with fallback: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	if ln.Addr().String() == occupied.Addr().String() {
		t.Fatalf("listener did not move ports (got %q)", ln.Addr().String())
	}
}
