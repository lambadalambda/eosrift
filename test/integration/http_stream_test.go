//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/client"
)

func TestHTTPTunnel_StreamingResponse_IsNotBuffered(t *testing.T) {
	t.Parallel()

	upstream := http.NewServeMux()
	upstream.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("chunk1\n"))
		fl.Flush()

		time.Sleep(1500 * time.Millisecond)

		_, _ = w.Write([]byte("chunk2\n"))
		fl.Flush()
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{
		Handler:           upstream,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(tunnelCtx, controlURL(), ln.Addr().String(), client.HTTPTunnelOptions{
		Authtoken: getenv("EOSRIFT_AUTHTOKEN", ""),
	})
	if err != nil {
		t.Fatalf("start http tunnel: %v", err)
	}
	defer tunnel.Close()

	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, httpURL("/stream"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = fmt.Sprintf("%s.tunnel.eosrift.test", tunnel.ID)

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d (body=%q)", resp.StatusCode, http.StatusOK, string(body))
	}

	started := time.Now()

	type readResult struct {
		buf []byte
		err error
	}
	first := make(chan readResult, 1)
	go func() {
		b := make([]byte, len("chunk1\n"))
		_, err := io.ReadFull(resp.Body, b)
		first <- readResult{buf: b, err: err}
	}()

	select {
	case r := <-first:
		if r.err != nil {
			t.Fatalf("read first chunk: %v", r.err)
		}
		if got, want := string(r.buf), "chunk1\n"; got != want {
			t.Fatalf("first chunk = %q, want %q", got, want)
		}
		if elapsed := time.Since(started); elapsed > 1*time.Second {
			t.Fatalf("first chunk took %s, want <= %s (buffered response?)", elapsed, 1*time.Second)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("did not receive first chunk within 1s (buffered response?)")
	}

	rest, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read rest: %v", err)
	}

	if got, want := string(rest), "chunk2\n"; got != want {
		t.Fatalf("rest = %q, want %q", got, want)
	}
}
