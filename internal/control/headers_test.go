package control

import "testing"

func TestNormalizeHeaderName(t *testing.T) {
	t.Parallel()

	t.Run("canonicalizes", func(t *testing.T) {
		t.Parallel()

		got, err := NormalizeHeaderName("x", "x-test")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got != "X-Test" {
			t.Fatalf("got = %q, want %q", got, "X-Test")
		}
	})

	t.Run("rejects disallowed", func(t *testing.T) {
		t.Parallel()

		if _, err := NormalizeHeaderName("x", "Host"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}

func TestValidateHeaderValue(t *testing.T) {
	t.Parallel()

	t.Run("trims", func(t *testing.T) {
		t.Parallel()

		got, err := ValidateHeaderValue("x", "X-Test", " ok ")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got != "ok" {
			t.Fatalf("got = %q, want %q", got, "ok")
		}
	})

	t.Run("rejects newline injection", func(t *testing.T) {
		t.Parallel()

		if _, err := ValidateHeaderValue("x", "X-Test", "ok\r\nX-Evil: 1"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}
