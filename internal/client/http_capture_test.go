package client

import "testing"

func TestPreviewCapture_TruncatesToLimit(t *testing.T) {
	t.Parallel()

	c := newPreviewCapture(3)
	_, _ = c.Write([]byte("hello"))

	if got, want := string(c.Bytes()), "hel"; got != want {
		t.Fatalf("bytes = %q, want %q", got, want)
	}
}

func TestPreviewCapture_ZeroLimitCapturesNothing(t *testing.T) {
	t.Parallel()

	c := newPreviewCapture(0)
	_, _ = c.Write([]byte("hello"))

	if got := len(c.Bytes()); got != 0 {
		t.Fatalf("len = %d, want %d", got, 0)
	}
}
