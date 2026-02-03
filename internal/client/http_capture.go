package client

import (
	"bytes"
	"context"
	"io"
	"net"
	"time"
)

type previewCapture struct {
	limit int
	buf   bytes.Buffer
}

func newPreviewCapture(limit int) *previewCapture {
	if limit < 0 {
		limit = 0
	}
	return &previewCapture{limit: limit}
}

func (c *previewCapture) Write(p []byte) (int, error) {
	if c.limit > 0 && c.buf.Len() < c.limit {
		remaining := c.limit - c.buf.Len()
		if remaining > 0 {
			n := len(p)
			if n > remaining {
				n = remaining
			}
			_, _ = c.buf.Write(p[:n])
		}
	}
	return len(p), nil
}

func (c *previewCapture) Bytes() []byte {
	return c.buf.Bytes()
}

type copyResult struct {
	dir int
	n   int64
	err error
}

const (
	copyDirRequest = iota
	copyDirResponse
)

func proxyBidirectionalWithCapture(ctx context.Context, upstream, stream net.Conn, reqCap, respCap io.Writer) (bytesIn, bytesOut int64, err error) {
	resCh := make(chan copyResult, 2)

	go func() {
		n, e := io.Copy(upstream, io.TeeReader(stream, reqCap))
		resCh <- copyResult{dir: copyDirRequest, n: n, err: e}
	}()
	go func() {
		n, e := io.Copy(stream, io.TeeReader(upstream, respCap))
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
