//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/cli"
	"eosrift.com/eosrift/internal/config"
)

func TestStart_All_StartsHTTPTunnelAndTCPTunnel(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello-from-start\n"))
	})

	httpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen http: %v", err)
	}
	t.Cleanup(func() { _ = httpLn.Close() })

	httpSrv := &http.Server{
		Handler:           upstream,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = httpSrv.Serve(httpLn) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	})

	tcpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	t.Cleanup(func() { _ = tcpLn.Close() })

	go func() {
		for {
			conn, err := tcpLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "eosrift.yml")

	sub := randomSubdomain(t, "start")
	if err := config.Save(cfgPath, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"db":  {Proto: "tcp", Addr: tcpLn.Addr().String()},
			"web": {Proto: "http", Addr: httpLn.Addr().String(), Domain: sub + ".tunnel.eosrift.test"},
		},
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdout, stderr lockedBuffer
	done := make(chan int, 1)
	go func() {
		done <- cli.Run(ctx, []string{
			"--config", cfgPath,
			"start",
			"--all",
			"--inspect=false",
			"--server", httpBaseURL(),
		}, &stdout, &stderr)
	}()

	out := waitForOutput(t, &stdout, done, []string{"Session Status", "Forwarding", "tcp://", "https://"})
	port := parseRemotePort(t, out)
	host := parseHTTPForwardingHost(t, out)

	assertHTTPTunnel(t, host, "hello-from-start\n")
	assertTCPTunnel(t, port)

	cancel()

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stdout=%q stderr=%q)", code, stdout.String(), stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("cli did not exit before timeout (stdout=%q stderr=%q)", stdout.String(), stderr.String())
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitForOutput(t *testing.T, stdout *lockedBuffer, done <-chan int, parts []string) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case code := <-done:
			t.Fatalf("cli exited early: code=%d (stdout=%q)", code, stdout.String())
		default:
		}

		out := stdout.String()
		ok := true
		for _, p := range parts {
			if !strings.Contains(out, p) {
				ok = false
				break
			}
		}
		if ok {
			return out
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for cli output (stdout=%q)", stdout.String())
	return ""
}

func parseRemotePort(t *testing.T, out string) int {
	t.Helper()

	re := regexp.MustCompile(`Forwarding\s+tcp://[^:]+:(\d+)`)
	m := re.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("missing tcp forwarding line (stdout=%q)", out)
	}
	port, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("parse tcp port: %v (stdout=%q)", err, out)
	}
	return port
}

func parseHTTPForwardingHost(t *testing.T, out string) string {
	t.Helper()

	re := regexp.MustCompile(`Forwarding\s+https?://([^\s]+)`)
	m := re.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("missing http forwarding line (stdout=%q)", out)
	}
	host := strings.TrimSpace(m[1])
	host = strings.TrimSuffix(host, "/")
	if host == "" {
		t.Fatalf("empty http forwarding host (stdout=%q)", out)
	}
	return host
}

func assertHTTPTunnel(t *testing.T, host, wantBody string) {
	t.Helper()

	clientHTTP := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, httpURL("/hello"), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Host = host

		resp, err := clientHTTP.Do(req)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if got := string(body); got != wantBody {
			t.Fatalf("http body = %q, want %q", got, wantBody)
		}
		return
	}

	t.Fatalf("http tunnel did not respond before timeout (host=%q)", host)
}

func assertTCPTunnel(t *testing.T, remotePort int) {
	t.Helper()

	addr := fmt.Sprintf("%s:%d", tcpDialHost(), remotePort)
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		defer conn.Close()

		msg := []byte("hello-start")
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

	t.Fatalf("tcp tunnel did not accept connections before timeout (addr=%q)", addr)
}

func randomSubdomain(t *testing.T, prefix string) string {
	t.Helper()

	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
