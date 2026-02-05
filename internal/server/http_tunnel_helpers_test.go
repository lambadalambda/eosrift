package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestApplyHeaderTransforms(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("X-Remove", "bye")
	h.Set("X-Keep", "old")

	applyHeaderTransforms(h,
		[]string{" X-Remove ", "", "X-Does-Not-Exist"},
		[]headerKV{
			{Name: " X-Add ", Value: "yes"},
			{Name: "", Value: "no"},
			{Name: "X-Keep", Value: "new"},
		},
	)

	if got := h.Get("X-Remove"); got != "" {
		t.Fatalf("X-Remove = %q, want empty", got)
	}
	if got := h.Get("X-Add"); got != "yes" {
		t.Fatalf("X-Add = %q, want %q", got, "yes")
	}
	if got := h.Get("X-Keep"); got != "new" {
		t.Fatalf("X-Keep = %q, want %q", got, "new")
	}
}

func TestCopyProxyForwardedHeaders(t *testing.T) {
	t.Parallel()

	src := make(http.Header)
	src["Forwarded"] = []string{"for=1.2.3.4"}
	src["X-Forwarded-For"] = []string{"1.2.3.4, 5.6.7.8"}
	src["X-Forwarded-Host"] = []string{"example.com"}
	src["X-Forwarded-Proto"] = []string{"https"}
	src["X-Other"] = []string{"ok"}

	dst := make(http.Header)
	copyProxyForwardedHeaders(dst, src)

	// Ensure values were copied (not aliased).
	src["Forwarded"][0] = "for=9.9.9.9"
	if got := dst.Get("Forwarded"); got != "for=1.2.3.4" {
		t.Fatalf("Forwarded = %q, want %q", got, "for=1.2.3.4")
	}

	for _, k := range []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
		if dst.Get(k) == "" {
			t.Fatalf("missing %s in dst", k)
		}
	}
	if got := dst.Get("X-Other"); got != "" {
		t.Fatalf("X-Other copied unexpectedly: %q", got)
	}
}

func TestStripForwardedHeaders(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("Forwarded", "for=1.2.3.4")
	h.Set("X-Forwarded-For", "1.2.3.4")
	h.Set("X-Forwarded-Host", "example.com")
	h.Set("X-Forwarded-Proto", "https")
	h.Set("X-Forwarded-Port", "443")
	h.Set("X-Real-IP", "1.2.3.4")
	h.Set("X-Keep", "ok")

	stripForwardedHeaders(h)

	for _, k := range []string{
		"Forwarded",
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"X-Forwarded-Proto",
		"X-Forwarded-Port",
		"X-Real-IP",
	} {
		if got := h.Get(k); got != "" {
			t.Fatalf("%s = %q, want empty", k, got)
		}
	}
	if got := h.Get("X-Keep"); got != "ok" {
		t.Fatalf("X-Keep = %q, want %q", got, "ok")
	}
}

func TestRequestClientIP(t *testing.T) {
	t.Parallel()

	t.Run("trust proxy headers disabled: ignores xff/xrealip", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		req.Header.Set("X-Real-IP", "1.2.3.4")

		got, ok := requestClientIP(req, false)
		if !ok {
			t.Fatalf("ok = false, want true")
		}
		if got.String() != "5.6.7.8" {
			t.Fatalf("got = %q, want %q", got, "5.6.7.8")
		}
	})

	t.Run("trust proxy headers enabled: uses xff", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		req.Header.Set("X-Forwarded-For", "1.2.3.4, 9.9.9.9")

		got, ok := requestClientIP(req, true)
		if !ok {
			t.Fatalf("ok = false, want true")
		}
		if got.String() != "1.2.3.4" {
			t.Fatalf("got = %q, want %q", got, "1.2.3.4")
		}
	})

	t.Run("trust proxy headers enabled: invalid xff falls back to x-real-ip", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		req.Header.Set("X-Forwarded-For", "nope")
		req.Header.Set("X-Real-IP", "1.2.3.4")

		got, ok := requestClientIP(req, true)
		if !ok {
			t.Fatalf("ok = false, want true")
		}
		if got.String() != "1.2.3.4" {
			t.Fatalf("got = %q, want %q", got, "1.2.3.4")
		}
	})

	t.Run("parses bracketed ipv6 remote addr", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.RemoteAddr = "[2001:db8::1]:443"

		got, ok := requestClientIP(req, false)
		if !ok {
			t.Fatalf("ok = false, want true")
		}
		if got.String() != "2001:db8::1" {
			t.Fatalf("got = %q, want %q", got, "2001:db8::1")
		}
	})

	t.Run("rejects invalid remote addr", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.RemoteAddr = "not-an-ip"

		if _, ok := requestClientIP(req, false); ok {
			t.Fatalf("ok = true, want false")
		}
	})
}

func TestCIDRListContains(t *testing.T) {
	t.Parallel()

	list := []netip.Prefix{
		netip.MustParsePrefix("1.2.3.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}

	if !cidrListContains(list, netip.MustParseAddr("1.2.3.4")) {
		t.Fatalf("expected 1.2.3.4 to match 1.2.3.0/24")
	}
	if cidrListContains(list, netip.MustParseAddr("5.6.7.8")) {
		t.Fatalf("expected 5.6.7.8 to not match any prefix")
	}
	if !cidrListContains(list, netip.MustParseAddr("2001:db8::1")) {
		t.Fatalf("expected 2001:db8::1 to match 2001:db8::/32")
	}
}
