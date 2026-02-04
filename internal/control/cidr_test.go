package control

import "testing"

func TestParseCIDRList(t *testing.T) {
	t.Parallel()

	t.Run("accepts CIDR", func(t *testing.T) {
		t.Parallel()

		got, err := ParseCIDRList("allow_cidr", []string{"1.2.3.0/24"}, 64)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0].String() != "1.2.3.0/24" {
			t.Fatalf("got = %#v, want %#v", got, []string{"1.2.3.0/24"})
		}
	})

	t.Run("accepts bare ip", func(t *testing.T) {
		t.Parallel()

		got, err := ParseCIDRList("allow_cidr", []string{"1.2.3.4"}, 64)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0].String() != "1.2.3.4/32" {
			t.Fatalf("got = %#v, want %#v", got, []string{"1.2.3.4/32"})
		}
	})

	t.Run("rejects invalid", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseCIDRList("allow_cidr", []string{"nope"}, 64); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})

	t.Run("enforces max entries", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseCIDRList("allow_cidr", []string{"1.2.3.0/24", "1.2.3.4"}, 1); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}
