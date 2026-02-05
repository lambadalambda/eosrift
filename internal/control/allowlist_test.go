package control

import "testing"

func TestParseHTTPMethodList(t *testing.T) {
	t.Parallel()

	t.Run("normalizes and uppercases", func(t *testing.T) {
		t.Parallel()

		got, err := ParseHTTPMethodList("allow_method", []string{" get "}, 64)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0] != "GET" {
			t.Fatalf("got = %#v, want %#v", got, []string{"GET"})
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseHTTPMethodList("allow_method", []string{"G ET"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects control chars", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseHTTPMethodList("allow_method", []string{"GET\r\n"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("enforces max entries", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseHTTPMethodList("allow_method", []string{"GET", "POST"}, 1); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}

func TestParsePathList(t *testing.T) {
	t.Parallel()

	t.Run("accepts absolute paths", func(t *testing.T) {
		t.Parallel()

		got, err := ParsePathList("allow_path", []string{" /healthz "}, 64)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0] != "/healthz" {
			t.Fatalf("got = %#v, want %#v", got, []string{"/healthz"})
		}
	})

	t.Run("rejects missing leading slash", func(t *testing.T) {
		t.Parallel()

		if _, err := ParsePathList("allow_path", []string{"healthz"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects query fragments", func(t *testing.T) {
		t.Parallel()

		if _, err := ParsePathList("allow_path", []string{"/x?y=1"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects whitespace", func(t *testing.T) {
		t.Parallel()

		if _, err := ParsePathList("allow_path", []string{"/bad path"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("enforces max entries", func(t *testing.T) {
		t.Parallel()

		if _, err := ParsePathList("allow_path", []string{"/a", "/b"}, 1); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}

