package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStore_ReserveSubdomain_Lifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()

	s, err := Open(ctx, filepath.Join(dir, "eosrift.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	rec, _, err := s.CreateToken(ctx, "owner")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := s.ReserveSubdomain(ctx, rec.ID, "demo"); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	gotID, ok, err := s.ReservedSubdomainTokenID(ctx, "demo")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !ok {
		t.Fatalf("lookup ok = false, want true")
	}
	if gotID != rec.ID {
		t.Fatalf("token id = %d, want %d", gotID, rec.ID)
	}

	if err := s.UnreserveSubdomain(ctx, "demo"); err != nil {
		t.Fatalf("unreserve: %v", err)
	}

	_, ok, err = s.ReservedSubdomainTokenID(ctx, "demo")
	if err != nil {
		t.Fatalf("lookup after delete: %v", err)
	}
	if ok {
		t.Fatalf("lookup ok = true after delete, want false")
	}
}

func TestStore_ReserveSubdomain_UniqueName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()

	s, err := Open(ctx, filepath.Join(dir, "eosrift.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	a, _, err := s.CreateToken(ctx, "a")
	if err != nil {
		t.Fatalf("create token a: %v", err)
	}
	b, _, err := s.CreateToken(ctx, "b")
	if err != nil {
		t.Fatalf("create token b: %v", err)
	}

	if err := s.ReserveSubdomain(ctx, a.ID, "demo"); err != nil {
		t.Fatalf("reserve a: %v", err)
	}
	if err := s.ReserveSubdomain(ctx, b.ID, "demo"); err == nil {
		t.Fatalf("reserve b: expected error")
	}
}

func TestNormalizeSubdomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		in         string
		wantOut    string
		wantErr    bool
	}{
		{"basic", "demo", "demo", false},
		{"trim+lower", "  DeMo  ", "demo", false},
		{"empty", "", "", true},
		{"dot", "a.b", "", true},
		{"start hyphen", "-demo", "", true},
		{"end hyphen", "demo-", "", true},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", "", true},
		{"invalid char", "nope!", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeSubdomain(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v (got=%q)", err, tc.wantErr, got)
			}
			if got != tc.wantOut {
				t.Fatalf("out = %q, want %q", got, tc.wantOut)
			}
		})
	}
}

