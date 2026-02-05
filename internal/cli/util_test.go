package cli

import (
	"net/http"
	"testing"
)

func TestControlHost(t *testing.T) {
	t.Parallel()

	t.Run("parses hostname", func(t *testing.T) {
		t.Parallel()

		got := controlHost("wss://eosrift.com/control")
		if got != "eosrift.com" {
			t.Fatalf("got = %q, want %q", got, "eosrift.com")
		}
	})

	t.Run("falls back on parse error", func(t *testing.T) {
		t.Parallel()

		in := "://bad"
		got := controlHost(in)
		if got != in {
			t.Fatalf("got = %q, want %q", got, in)
		}
	})
}

func TestStripHopByHopHeaders(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("Connection", "close")
	h.Set("Proxy-Connection", "keep-alive")
	h.Set("Keep-Alive", "timeout=5")
	h.Set("Upgrade", "websocket")
	h.Set("X-Test", "ok")

	stripHopByHopHeaders(h)

	for _, k := range []string{"Connection", "Proxy-Connection", "Keep-Alive", "Upgrade"} {
		if v := h.Get(k); v != "" {
			t.Fatalf("%s = %q, want empty", k, v)
		}
	}
	if got := h.Get("X-Test"); got != "ok" {
		t.Fatalf("X-Test = %q, want %q", got, "ok")
	}
}
