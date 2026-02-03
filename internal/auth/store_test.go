package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStore_CreateListRevokeValidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()

	s, err := Open(ctx, filepath.Join(dir, "eosrift.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	rec, token, err := s.CreateToken(ctx, "test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if token == "" {
		t.Fatalf("token empty")
	}
	if rec.ID == 0 {
		t.Fatalf("id = %d, want non-zero", rec.ID)
	}
	if rec.Label != "test" {
		t.Fatalf("label = %q, want %q", rec.Label, "test")
	}
	if rec.Prefix == "" {
		t.Fatalf("prefix empty")
	}
	if rec.RevokedAt != nil {
		t.Fatalf("revoked_at = %v, want nil", rec.RevokedAt)
	}

	ok, err := s.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !ok {
		t.Fatalf("validate ok = false, want true")
	}

	list, err := s.ListTokens(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want %d", len(list), 1)
	}
	if list[0].ID != rec.ID {
		t.Fatalf("list id = %d, want %d", list[0].ID, rec.ID)
	}
	if list[0].Label != "test" {
		t.Fatalf("list label = %q, want %q", list[0].Label, "test")
	}
	if list[0].Prefix != rec.Prefix {
		t.Fatalf("list prefix = %q, want %q", list[0].Prefix, rec.Prefix)
	}
	if list[0].RevokedAt != nil {
		t.Fatalf("list revoked_at = %v, want nil", list[0].RevokedAt)
	}

	if err := s.RevokeToken(ctx, rec.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	ok, err = s.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("validate after revoke: %v", err)
	}
	if ok {
		t.Fatalf("validate ok = true after revoke, want false")
	}
}

func TestStore_EnsureTokenIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()

	s, err := Open(ctx, filepath.Join(dir, "eosrift.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.EnsureToken(ctx, "eos_test_token", "bootstrap"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := s.EnsureToken(ctx, "eos_test_token", "bootstrap"); err != nil {
		t.Fatalf("ensure second: %v", err)
	}

	list, err := s.ListTokens(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want %d", len(list), 1)
	}

	ok, err := s.ValidateToken(ctx, "eos_test_token")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !ok {
		t.Fatalf("validate ok = false, want true")
	}
}
