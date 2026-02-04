package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/config"
)

func TestRun_StartAllAndNames_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "start", "--all", "web"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--all cannot be combined") {
		t.Fatalf("stderr missing --all error: %q", stderr.String())
	}
}

func TestRun_Start_NoConfigFile_Errors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "start", "--all"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "config file not found") {
		t.Fatalf("stderr missing config not found: %q", stderr.String())
	}
}

func TestRun_Start_NoTunnelsInConfig_Errors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{Version: 1}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "start", "--all"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no tunnels defined") {
		t.Fatalf("stderr missing no tunnels defined: %q", stderr.String())
	}
}

func TestRun_Start_UnknownTunnel_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"web": {Proto: "http", Addr: "3000"},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "start", "db"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown tunnel") {
		t.Fatalf("stderr missing unknown tunnel: %q", stderr.String())
	}
}
