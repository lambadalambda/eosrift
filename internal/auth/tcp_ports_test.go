package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStore_ReserveTCPPort_Lifecycle(t *testing.T) {
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

	if err := s.ReserveTCPPort(ctx, rec.ID, 20005); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	gotID, ok, err := s.ReservedTCPPortTokenID(ctx, 20005)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !ok {
		t.Fatalf("lookup ok = false, want true")
	}
	if gotID != rec.ID {
		t.Fatalf("token id = %d, want %d", gotID, rec.ID)
	}

	if err := s.UnreserveTCPPort(ctx, 20005); err != nil {
		t.Fatalf("unreserve: %v", err)
	}

	_, ok, err = s.ReservedTCPPortTokenID(ctx, 20005)
	if err != nil {
		t.Fatalf("lookup after delete: %v", err)
	}
	if ok {
		t.Fatalf("lookup ok = true after delete, want false")
	}
}

func TestStore_ReserveTCPPort_UniquePort(t *testing.T) {
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

	if err := s.ReserveTCPPort(ctx, a.ID, 20005); err != nil {
		t.Fatalf("reserve a: %v", err)
	}
	if err := s.ReserveTCPPort(ctx, b.ID, 20005); err == nil {
		t.Fatalf("reserve b: expected error")
	}
}

func TestStore_ListReservedTCPPorts(t *testing.T) {
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

	if err := s.ReserveTCPPort(ctx, rec.ID, 20006); err != nil {
		t.Fatalf("reserve 20006: %v", err)
	}
	if err := s.ReserveTCPPort(ctx, rec.ID, 20005); err != nil {
		t.Fatalf("reserve 20005: %v", err)
	}

	list, err := s.ListReservedTCPPorts(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want %d", len(list), 2)
	}
	if list[0].Port != 20005 || list[1].Port != 20006 {
		t.Fatalf("ports = %d, %d, want %d, %d", list[0].Port, list[1].Port, 20005, 20006)
	}
	for _, r := range list {
		if r.TokenID != rec.ID {
			t.Fatalf("token id = %d, want %d", r.TokenID, rec.ID)
		}
		if r.TokenPrefix == "" {
			t.Fatalf("token prefix empty")
		}
		if r.CreatedAt.IsZero() {
			t.Fatalf("created_at is zero")
		}
	}
}
