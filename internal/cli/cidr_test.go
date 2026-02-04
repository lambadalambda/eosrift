package cli

import "testing"

func TestValidateCIDRs(t *testing.T) {
	t.Parallel()

	t.Run("accepts cidr", func(t *testing.T) {
		t.Parallel()

		if err := validateCIDRs("allow_cidr", []string{"1.2.3.0/24"}); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("accepts bare ip", func(t *testing.T) {
		t.Parallel()

		if err := validateCIDRs("allow_cidr", []string{"1.2.3.4"}); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("rejects invalid", func(t *testing.T) {
		t.Parallel()

		if err := validateCIDRs("allow_cidr", []string{"nope"}); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
	})
}
