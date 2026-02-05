package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/auth"
)

func TestRunTokenCmd_NoArgs_ShowsUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := runTokenCmd(nil, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage: eosrift-server token") {
		t.Fatalf("stderr missing usage: %q", stderr.String())
	}
}

func TestRunTokenCmd_CreateListRevoke(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "eosrift.db")

	var createOut, createErr bytes.Buffer
	code := runTokenCmd(nil, []string{"create", "--db", dbPath, "--label", "testlabel"}, &createOut, &createErr)
	if code != 0 {
		t.Fatalf("create code = %d, want 0 (stderr=%q)", code, createErr.String())
	}
	if createErr.Len() != 0 {
		t.Fatalf("create stderr not empty: %q", createErr.String())
	}

	tokenID, token := parseTokenCreateOutput(t, createOut.String())

	ctx := context.Background()
	store, err := auth.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ok, err := store.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if !ok {
		t.Fatalf("ValidateToken = false, want true")
	}

	var listOut, listErr bytes.Buffer
	code = runTokenCmd(nil, []string{"list", "--db", dbPath}, &listOut, &listErr)
	if code != 0 {
		t.Fatalf("list code = %d, want 0 (stderr=%q)", code, listErr.String())
	}
	if listErr.Len() != 0 {
		t.Fatalf("list stderr not empty: %q", listErr.String())
	}
	if !strings.Contains(listOut.String(), "testlabel") {
		t.Fatalf("list output missing label: %q", listOut.String())
	}

	var revokeOut, revokeErr bytes.Buffer
	code = runTokenCmd(nil, []string{"revoke", "--db", dbPath, strconv.FormatInt(tokenID, 10)}, &revokeOut, &revokeErr)
	if code != 0 {
		t.Fatalf("revoke code = %d, want 0 (stderr=%q)", code, revokeErr.String())
	}
	if revokeErr.Len() != 0 {
		t.Fatalf("revoke stderr not empty: %q", revokeErr.String())
	}

	ok, err = store.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if ok {
		t.Fatalf("ValidateToken = true, want false")
	}
}

func TestRunReserveCmd_AddListRemove(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "eosrift.db")

	var createOut bytes.Buffer
	if code := runTokenCmd(nil, []string{"create", "--db", dbPath}, &createOut, &bytes.Buffer{}); code != 0 {
		t.Fatalf("create token failed")
	}
	tokenID, _ := parseTokenCreateOutput(t, createOut.String())

	var stdout, stderr bytes.Buffer
	code := runReserveCmd(nil, []string{"add", "--db", dbPath, "--token-id", strconv.FormatInt(tokenID, 10), "demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reserve add code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	gotTokenID, ok, err := store.ReservedSubdomainTokenID(ctx, "demo")
	if err != nil {
		t.Fatalf("ReservedSubdomainTokenID: %v", err)
	}
	if !ok || gotTokenID != tokenID {
		t.Fatalf("ReservedSubdomainTokenID = (%d, %v), want (%d, true)", gotTokenID, ok, tokenID)
	}

	stdout.Reset()
	stderr.Reset()
	code = runReserveCmd(nil, []string{"list", "--db", dbPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reserve list code = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo\t") {
		t.Fatalf("list output missing demo: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runReserveCmd(nil, []string{"remove", "--db", dbPath, "demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reserve remove code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	_, ok, err = store.ReservedSubdomainTokenID(ctx, "demo")
	if err != nil {
		t.Fatalf("ReservedSubdomainTokenID: %v", err)
	}
	if ok {
		t.Fatalf("reservation still exists, want removed")
	}
}

func TestRunTCPReserveCmd_AddListRemove(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "eosrift.db")

	var createOut bytes.Buffer
	if code := runTokenCmd(nil, []string{"create", "--db", dbPath}, &createOut, &bytes.Buffer{}); code != 0 {
		t.Fatalf("create token failed")
	}
	tokenID, _ := parseTokenCreateOutput(t, createOut.String())

	var stdout, stderr bytes.Buffer
	code := runTCPReserveCmd(nil, []string{"add", "--db", dbPath, "--token-id", strconv.FormatInt(tokenID, 10), "12345"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("tcp-reserve add code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	gotTokenID, ok, err := store.ReservedTCPPortTokenID(ctx, 12345)
	if err != nil {
		t.Fatalf("ReservedTCPPortTokenID: %v", err)
	}
	if !ok || gotTokenID != tokenID {
		t.Fatalf("ReservedTCPPortTokenID = (%d, %v), want (%d, true)", gotTokenID, ok, tokenID)
	}

	stdout.Reset()
	stderr.Reset()
	code = runTCPReserveCmd(nil, []string{"list", "--db", dbPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("tcp-reserve list code = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "12345\t") {
		t.Fatalf("list output missing 12345: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runTCPReserveCmd(nil, []string{"remove", "--db", dbPath, "12345"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("tcp-reserve remove code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	_, ok, err = store.ReservedTCPPortTokenID(ctx, 12345)
	if err != nil {
		t.Fatalf("ReservedTCPPortTokenID: %v", err)
	}
	if ok {
		t.Fatalf("tcp reservation still exists, want removed")
	}
}

func parseTokenCreateOutput(t *testing.T, out string) (int64, string) {
	t.Helper()

	var id int64
	var token string

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "id:") {
			n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "id:")), 10, 64)
			if err != nil {
				t.Fatalf("parse id: %v (line=%q)", err, line)
			}
			id = n
			continue
		}
		if strings.HasPrefix(line, "token:") {
			token = strings.TrimSpace(strings.TrimPrefix(line, "token:"))
		}
	}

	if id <= 0 {
		t.Fatalf("missing id in output: %q", out)
	}
	if token == "" {
		t.Fatalf("missing token in output: %q", out)
	}

	return id, token
}
