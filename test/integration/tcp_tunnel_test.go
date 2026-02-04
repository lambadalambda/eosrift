//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
)

func TestTCPTunnel_Echo(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartTCPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.TCPTunnelOptions{
		Authtoken: getenv("EOSRIFT_AUTHTOKEN", ""),
	})
	if err != nil {
		t.Fatalf("start tcp tunnel: %v", err)
	}
	defer tunnel.Close()

	if tunnel.RemotePort < 20000 || tunnel.RemotePort > 20010 {
		t.Fatalf("remote port = %d, want within [20000,20010]", tunnel.RemotePort)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", tcpDialHost(), tunnel.RemotePort), 5*time.Second)
	if err != nil {
		t.Fatalf("dial remote: %v", err)
	}
	defer conn.Close()

	msg := []byte("hello-eosrift")
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(got) != string(msg) {
		t.Fatalf("echo = %q, want %q", string(got), string(msg))
	}
}

func TestTCPTunnel_RequestRemotePort_HonorsRequest(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for port := 20000; port <= 20010; port++ {
		tunnel, err := client.StartTCPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.TCPTunnelOptions{
			Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
			RemotePort: port,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "requested port unavailable") {
				continue
			}
			t.Fatalf("start tcp tunnel (remote port=%d): %v", port, err)
		}
		defer tunnel.Close()

		if tunnel.RemotePort != port {
			t.Fatalf("remote port = %d, want %d", tunnel.RemotePort, port)
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", tcpDialHost(), tunnel.RemotePort), 5*time.Second)
		if err != nil {
			t.Fatalf("dial remote: %v", err)
		}
		defer conn.Close()

		msg := []byte("hello-eosrift")
		if _, err := conn.Write(msg); err != nil {
			t.Fatalf("write: %v", err)
		}

		got := make([]byte, len(msg))
		if _, err := io.ReadFull(conn, got); err != nil {
			t.Fatalf("read: %v", err)
		}

		if string(got) != string(msg) {
			t.Fatalf("echo = %q, want %q", string(got), string(msg))
		}

		return
	}

	t.Fatalf("no available remote port in range [20000,20010]")
}

func TestTCPTunnel_RequestRemotePort_OutOfRange_Errors(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartTCPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.TCPTunnelOptions{
		Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
		RemotePort: 19999,
	})
	if err == nil {
		_ = tunnel.Close()
		t.Fatalf("expected error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "requested port out of range") {
		t.Fatalf("error = %q, want requested port out of range", err.Error())
	}
}

func TestTCPTunnel_RequestRemotePort_Unavailable_Errors(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for port := 20000; port <= 20010; port++ {
		tunnel1, err := client.StartTCPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.TCPTunnelOptions{
			Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
			RemotePort: port,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "requested port unavailable") {
				continue
			}
			t.Fatalf("start tcp tunnel (remote port=%d): %v", port, err)
		}
		defer tunnel1.Close()

		tunnel2, err := client.StartTCPTunnelWithOptions(ctx, controlURL(), ln.Addr().String(), client.TCPTunnelOptions{
			Authtoken:  getenv("EOSRIFT_AUTHTOKEN", ""),
			RemotePort: port,
		})
		if err == nil {
			_ = tunnel2.Close()
			t.Fatalf("expected remote port unavailable error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "requested port unavailable") {
			t.Fatalf("error = %q, want requested port unavailable", err.Error())
		}

		return
	}

	t.Fatalf("no available remote port in range [20000,20010]")
}
