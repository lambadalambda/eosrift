package client

import "testing"

func TestSummarizeHTTPExchange(t *testing.T) {
	t.Parallel()

	req := []byte("GET /hello?x=1 HTTP/1.1\r\nHost: abcd.tunnel.eosrift.test\r\nUser-Agent: test\r\n\r\n")
	resp := []byte("HTTP/1.1 201 Created\r\nContent-Type: text/plain\r\nContent-Length: 2\r\n\r\nok")

	s, ok := summarizeHTTPExchange(req, resp)
	if !ok {
		t.Fatalf("ok = false, want true")
	}

	if s.Method != "GET" {
		t.Fatalf("method = %q, want %q", s.Method, "GET")
	}
	if s.Path != "/hello?x=1" {
		t.Fatalf("path = %q, want %q", s.Path, "/hello?x=1")
	}
	if s.Host != "abcd.tunnel.eosrift.test" {
		t.Fatalf("host = %q, want %q", s.Host, "abcd.tunnel.eosrift.test")
	}
	if s.StatusCode != 201 {
		t.Fatalf("status = %d, want %d", s.StatusCode, 201)
	}
	if got := s.RequestHeaders.Get("User-Agent"); got != "test" {
		t.Fatalf("request user-agent = %q, want %q", got, "test")
	}
	if got := s.ResponseHeaders.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("response content-type = %q, want %q", got, "text/plain")
	}
}

func TestSummarizeHTTPExchange_Invalid(t *testing.T) {
	t.Parallel()

	if _, ok := summarizeHTTPExchange([]byte("not-http"), []byte("also-not-http")); ok {
		t.Fatalf("ok = true, want false")
	}
}
