package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/config"
)

func TestRun_ConfigAddAuthtoken_WritesConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "add-authtoken", "tok_test"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}

	cfg, ok, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatalf("Load ok = false, want true")
	}
	if cfg.Authtoken != "tok_test" {
		t.Fatalf("authtoken = %q, want %q", cfg.Authtoken, "tok_test")
	}
}

func TestRun_ConfigCheck_WarnsForMissingValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{Version: 1}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config OK:") {
		t.Fatalf("stdout missing Config OK: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "warning: authtoken is empty") {
		t.Fatalf("stdout missing authtoken warning: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "warning: server_addr is empty") {
		t.Fatalf("stdout missing server_addr warning: %q", stdout.String())
	}
}

func TestRun_ConfigCheck_RejectsInvalidHostHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{Version: 1, HostHeader: "bad value"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "check"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if !strings.Contains(stderr.String(), "host-header") {
		t.Fatalf("stderr missing host-header error: %q", stderr.String())
	}
}

func TestRun_Config_UnknownSubcommand_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "nope"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown config command") {
		t.Fatalf("stderr missing unknown config command: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage: eosrift config") {
		t.Fatalf("stderr missing usage: %q", stderr.String())
	}
}
