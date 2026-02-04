package cli

import "testing"

func TestParseHeaderKV(t *testing.T) {
	t.Parallel()

	t.Run("colon", func(t *testing.T) {
		t.Parallel()

		kv, err := parseHeaderKV("request_header_add", "x-test: ok")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if kv.Name != "X-Test" {
			t.Fatalf("name = %q, want %q", kv.Name, "X-Test")
		}
		if kv.Value != "ok" {
			t.Fatalf("value = %q, want %q", kv.Value, "ok")
		}
	})

	t.Run("equals", func(t *testing.T) {
		t.Parallel()

		kv, err := parseHeaderKV("request_header_add", "x-test=ok")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if kv.Name != "X-Test" {
			t.Fatalf("name = %q, want %q", kv.Name, "X-Test")
		}
		if kv.Value != "ok" {
			t.Fatalf("value = %q, want %q", kv.Value, "ok")
		}
	})

	t.Run("rejects disallowed header", func(t *testing.T) {
		t.Parallel()

		if _, err := parseHeaderKV("request_header_add", "Host: example.com"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("rejects newline injection", func(t *testing.T) {
		t.Parallel()

		if _, err := parseHeaderKV("request_header_add", "X-Test: ok\r\nX-Evil: 1"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}

func TestParseHeaderRemoveList(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid", func(t *testing.T) {
		t.Parallel()

		got, err := parseHeaderRemoveList("request_header_remove", []string{"x-test"})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0] != "X-Test" {
			t.Fatalf("got = %#v, want %#v", got, []string{"X-Test"})
		}
	})

	t.Run("rejects hop-by-hop", func(t *testing.T) {
		t.Parallel()

		if _, err := parseHeaderRemoveList("request_header_remove", []string{"Connection"}); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}
