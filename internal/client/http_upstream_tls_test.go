package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDialHTTPUpstream_HTTPS_VerifyToggle(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := dialHTTPUpstream(ctx, "https", u.Host, true)
	if err != nil {
		t.Fatalf("dial (skip verify): %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Write([]byte("GET / HTTP/1.1\r\nHost: upstream\r\nConnection: close\r\n\r\n"))

	respBytes, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	resp := string(respBytes)

	if !strings.Contains(resp, "200") {
		t.Fatalf("response missing 200: %q", resp)
	}
	if !strings.Contains(resp, "ok\n") {
		t.Fatalf("response missing body: %q", resp)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	_, err = dialHTTPUpstream(ctx2, "https", u.Host, false)
	if err == nil {
		t.Fatalf("expected dial error with verification enabled")
	}
}
