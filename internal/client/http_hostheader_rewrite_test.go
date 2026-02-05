package client

import (
	"io"
	"net"
	"testing"
)

func TestRewriteHostHeader(t *testing.T) {
	t.Parallel()

	t.Run("replaces existing", func(t *testing.T) {
		t.Parallel()

		in := "GET /hello HTTP/1.1\r\nHost: old.example\r\nX-Test: ok\r\n\r\n"
		out := rewriteHostHeader([]byte(in), "new.example")
		want := "GET /hello HTTP/1.1\r\nHost: new.example\r\nX-Test: ok\r\n\r\n"
		if string(out) != want {
			t.Fatalf("out = %q, want %q", string(out), want)
		}
	})

	t.Run("adds if missing", func(t *testing.T) {
		t.Parallel()

		in := "GET /hello HTTP/1.1\r\nX-Test: ok\r\n\r\n"
		out := rewriteHostHeader([]byte(in), "new.example")
		want := "GET /hello HTTP/1.1\r\nX-Test: ok\r\nHost: new.example\r\n\r\n"
		if string(out) != want {
			t.Fatalf("out = %q, want %q", string(out), want)
		}
	})

	t.Run("empty mode is no-op", func(t *testing.T) {
		t.Parallel()

		in := "GET /hello HTTP/1.1\r\nHost: old.example\r\n\r\n"
		out := rewriteHostHeader([]byte(in), "  ")
		if string(out) != in {
			t.Fatalf("out = %q, want %q", string(out), in)
		}
	})
}

func TestCopyRequestWithHostRewrite(t *testing.T) {
	t.Parallel()

	upstreamWriter, upstreamReader := net.Pipe()
	streamReader, streamWriter := net.Pipe()

	errCh := make(chan error, 1)
	go func() {
		defer upstreamWriter.Close()
		defer streamReader.Close()

		_, err := copyRequestWithHostRewrite(upstreamWriter, streamReader, nil, "new.example")
		errCh <- err
	}()

	in := "GET /hello HTTP/1.1\r\nHost: old.example\r\nX-Test: ok\r\n\r\nBODY"
	if _, err := io.WriteString(streamWriter, in); err != nil {
		t.Fatalf("write stream: %v", err)
	}
	_ = streamWriter.Close()

	outBytes, err := io.ReadAll(upstreamReader)
	if err != nil {
		t.Fatalf("read upstream: %v", err)
	}
	_ = upstreamReader.Close()

	if err := <-errCh; err != nil && err != io.EOF {
		t.Fatalf("copy err = %v, want nil/EOF", err)
	}

	want := "GET /hello HTTP/1.1\r\nHost: new.example\r\nX-Test: ok\r\n\r\nBODY"
	if string(outBytes) != want {
		t.Fatalf("out = %q, want %q", string(outBytes), want)
	}
}
