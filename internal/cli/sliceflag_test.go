package cli

import "testing"

func TestStringSliceFlag(t *testing.T) {
	t.Parallel()

	t.Run("Set splits and trims", func(t *testing.T) {
		t.Parallel()

		var f stringSliceFlag
		if err := f.Set("a, b"); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got, want := []string(f), []string{"a", "b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("got = %#v, want %#v", got, want)
		}
		if got, want := f.String(), "a,b"; got != want {
			t.Fatalf("String() = %q, want %q", got, want)
		}
	})

	t.Run("Set rejects empty and does not mutate", func(t *testing.T) {
		t.Parallel()

		f := stringSliceFlag{"x"}
		if err := f.Set("a,"); err == nil {
			t.Fatalf("err = nil, want non-nil")
		}
		if got, want := []string(f), []string{"x"}; len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("got = %#v, want %#v", got, want)
		}
	})
}

func TestStringListFlag(t *testing.T) {
	t.Parallel()

	var f stringListFlag
	if err := f.Set("a"); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if err := f.Set("b"); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got, want := []string(f), []string{"a", "b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
	if got, want := f.String(), "a,b"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
