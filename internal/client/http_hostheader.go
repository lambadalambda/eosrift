package client

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"time"
)

func proxyBidirectionalWithHostRewrite(ctx context.Context, upstream, stream net.Conn, reqCap, respCap io.Writer, hostHeader string) (bytesIn, bytesOut int64, err error) {
	resCh := make(chan copyResult, 2)

	go func() {
		n, e := copyRequestWithHostRewrite(upstream, stream, reqCap, hostHeader)
		resCh <- copyResult{dir: copyDirRequest, n: n, err: e}
	}()
	go func() {
		var src io.Reader = upstream
		if respCap != nil {
			src = io.TeeReader(upstream, respCap)
		}
		n, e := io.Copy(stream, src)
		resCh <- copyResult{dir: copyDirResponse, n: n, err: e}
	}()

	var firstErr error
	deadlineSet := false
	doneCh := ctx.Done()

	received := 0
	for received < 2 {
		select {
		case <-doneCh:
			if !deadlineSet {
				_ = upstream.SetDeadline(time.Now())
				_ = stream.SetDeadline(time.Now())
				firstErr = ctx.Err()
				deadlineSet = true
			}
			doneCh = nil
		case res := <-resCh:
			received++
			if res.dir == copyDirRequest {
				bytesIn = res.n
			} else {
				bytesOut = res.n
			}
			if !deadlineSet {
				_ = upstream.SetDeadline(time.Now())
				_ = stream.SetDeadline(time.Now())
				firstErr = res.err
				deadlineSet = true
			}
		}
	}

	return bytesIn, bytesOut, firstErr
}

func copyRequestWithHostRewrite(upstream, stream net.Conn, reqCap io.Writer, hostHeader string) (int64, error) {
	hostHeader = strings.TrimSpace(hostHeader)
	if hostHeader == "" {
		return io.Copy(upstream, stream)
	}

	const maxHeaderBytes = 64 * 1024

	var (
		buf   bytes.Buffer
		tmp   = make([]byte, 4096)
		total int64
	)

	for {
		n, err := stream.Read(tmp)
		if n > 0 {
			chunk := tmp[:n]
			total += int64(n)

			if reqCap != nil {
				_, _ = reqCap.Write(chunk)
			}

			_, _ = buf.Write(chunk)
			if buf.Len() > maxHeaderBytes {
				// Header too large; fall back to raw proxying.
				if _, werr := upstream.Write(buf.Bytes()); werr != nil {
					return total, werr
				}
				buf.Reset()
				n2, e2 := io.Copy(upstream, stream)
				return total + n2, e2
			}

			b := buf.Bytes()
			if idx := bytes.Index(b, []byte("\r\n\r\n")); idx >= 0 {
				header := b[:idx+4]
				rest := b[idx+4:]

				outHeader := rewriteHostHeader(header, hostHeader)
				if _, werr := upstream.Write(outHeader); werr != nil {
					return total, werr
				}
				if len(rest) > 0 {
					if _, werr := upstream.Write(rest); werr != nil {
						return total, werr
					}
				}

				n2, e2 := io.Copy(upstream, stream)
				return total + n2, e2
			}
		}

		if err != nil {
			if buf.Len() > 0 {
				_, _ = upstream.Write(buf.Bytes())
			}
			return total, err
		}
	}
}

func rewriteHostHeader(header []byte, hostHeader string) []byte {
	hostHeader = strings.TrimSpace(hostHeader)
	if hostHeader == "" {
		return header
	}

	trimmed := header
	if bytes.HasSuffix(trimmed, []byte("\r\n\r\n")) {
		trimmed = trimmed[:len(trimmed)-4]
	}

	lines := bytes.Split(trimmed, []byte("\r\n"))
	if len(lines) == 0 {
		return header
	}

	var out bytes.Buffer
	out.Grow(len(header) + len(hostHeader) + 16)

	out.Write(lines[0])
	out.WriteString("\r\n")

	found := false
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			continue
		}

		lower := bytes.ToLower(line)
		if bytes.HasPrefix(lower, []byte("host:")) {
			out.WriteString("Host: ")
			out.WriteString(hostHeader)
			out.WriteString("\r\n")
			found = true
			continue
		}

		out.Write(line)
		out.WriteString("\r\n")
	}

	if !found {
		out.WriteString("Host: ")
		out.WriteString(hostHeader)
		out.WriteString("\r\n")
	}

	out.WriteString("\r\n")

	return out.Bytes()
}

