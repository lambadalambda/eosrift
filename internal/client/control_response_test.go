package client

import (
	"errors"
	"io"
	"testing"

	"eosrift.com/eosrift/internal/control"
)

type errAfterDataReadCloser struct {
	data []byte
	err  error
}

func (r *errAfterDataReadCloser) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return 0, io.EOF
}

func (r *errAfterDataReadCloser) Close() error { return nil }

func TestReadJSONControlResponse_ParsesPayloadOnTrailingReadError(t *testing.T) {
	t.Parallel()

	stream := &errAfterDataReadCloser{
		data: []byte(`{"type":"tcp","error":"requested port out of range"}` + "\n"),
		err:  errors.New("session shutdown"),
	}

	resp, err := readJSONControlResponse[control.CreateTCPTunnelResponse](stream)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if resp.Error != "requested port out of range" {
		t.Fatalf("resp.Error = %q, want %q", resp.Error, "requested port out of range")
	}
}

func TestReadJSONControlResponse_EmptyPayloadReturnsEOF(t *testing.T) {
	t.Parallel()

	stream := &errAfterDataReadCloser{}
	_, err := readJSONControlResponse[control.CreateTCPTunnelResponse](stream)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadJSONControlResponse_ReadErrorBeforePayload(t *testing.T) {
	t.Parallel()

	stream := &errAfterDataReadCloser{err: errors.New("session shutdown")}
	_, err := readJSONControlResponse[control.CreateTCPTunnelResponse](stream)
	if err == nil || err.Error() != "session shutdown" {
		t.Fatalf("err = %v, want %q", err, "session shutdown")
	}
}
