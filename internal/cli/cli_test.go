package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/config"
)

func TestRun_NoArgs_ShowsUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d", code, 2)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage: eosrift") {
		t.Fatalf("stderr missing usage: %q", stderr.String())
	}
}

func TestRun_Version(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if !strings.Contains(stdout.String(), "eosrift") {
		t.Fatalf("stdout missing version: %q", stdout.String())
	}
}

func TestRun_ConfigSetServer_WritesConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "set-server", "https://example.com"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}

	cfg, ok, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatalf("Load ok = false, want true")
	}
	if cfg.ServerAddr != "https://example.com" {
		t.Fatalf("server_addr = %q, want %q", cfg.ServerAddr, "https://example.com")
	}
}

func TestRun_ConfigSetHostHeader_WritesConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "set-host-header", "rewrite"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}

	cfg, ok, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatalf("Load ok = false, want true")
	}
	if cfg.HostHeader != "rewrite" {
		t.Fatalf("host_header = %q, want %q", cfg.HostHeader, "rewrite")
	}
}

func TestRun_HTTPHelp_PrintsToStdout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "http", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: eosrift http") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
}

func TestRun_HTTPHelp_UsesConfigHostHeaderDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")
	if err := config.Save(path, config.File{Version: 1, HostHeader: "rewrite"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "http", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}

	if !strings.Contains(stdout.String(), "Host header mode: preserve (default), rewrite, or a literal value (default \"rewrite\")") {
		t.Fatalf("stdout missing host-header default: %q", stdout.String())
	}
}

func TestRun_StartHelp_PrintsToStdout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "start", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: eosrift start") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
}

func TestRun_TCPHelp_PrintsToStdout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "tcp", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: eosrift tcp") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "remote-port") {
		t.Fatalf("stdout missing remote-port flag: %q", stdout.String())
	}
}

func TestRun_ConfigHelp_PrintsToStdout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "config", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: eosrift config") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
}
