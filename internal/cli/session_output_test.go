package cli

import (
	"bytes"
	"testing"
)

func TestPrintSession_HTTP_Golden(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSession(&buf, sessionOutput{
		Version:        "v0.1.0",
		Status:         "online",
		ForwardingFrom: "https://abc123.tunnel.eosrift.com",
		ForwardingTo:   "localhost:3000",
		Inspector:      "http://localhost:4040",
	})

	want := readGolden(t, "session_http.txt")
	if got := buf.String(); got != want {
		t.Fatalf("output mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestPrintSession_TCP_Golden(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSession(&buf, sessionOutput{
		Version:        "v0.1.0",
		Status:         "online",
		ForwardingFrom: "tcp://example.com:20001",
		ForwardingTo:   "localhost:5432",
	})

	want := readGolden(t, "session_tcp.txt")
	if got := buf.String(); got != want {
		t.Fatalf("output mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}
