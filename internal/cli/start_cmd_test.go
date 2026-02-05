package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRun_Start_HTTPDomainAndSubdomain_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"web": {Proto: "http", Addr: "3000", Domain: "demo.tunnel.eosrift.com", Subdomain: "demo"},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "web"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "only one of") {
		t.Fatalf("stderr missing domain/subdomain conflict: %q", stderr.String())
	}
}

func TestRun_Start_HTTPRemotePort_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"web": {Proto: "http", Addr: "3000", RemotePort: 20005},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "web"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "remote_port") {
		t.Fatalf("stderr missing remote_port error: %q", stderr.String())
	}
}

func TestRun_Start_TCPDomain_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"db": {Proto: "tcp", Addr: "5432", Domain: "demo.tunnel.eosrift.com"},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "db"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "domain") {
		t.Fatalf("stderr missing tcp domain error: %q", stderr.String())
	}
}

func TestRun_Start_InvalidAddr_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"web": {Proto: "http", Addr: "abc"},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "web"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "invalid addr") {
		t.Fatalf("stderr missing invalid addr error: %q", stderr.String())
	}
}

func TestRun_Start_HTTPInvalidAllowMethod_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"web": {Proto: "http", Addr: "3000", AllowMethod: []string{"G ET"}},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "web"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "allow_method") {
		t.Fatalf("stderr missing allow_method error: %q", stderr.String())
	}
}

func TestRun_Start_TCPAllowMethod_IsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := config.Save(path, config.File{
		Version: 1,
		Tunnels: map[string]config.Tunnel{
			"db": {Proto: "tcp", Addr: "5432", AllowMethod: []string{"GET"}},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := Run(ctx, []string{"--config", path, "start", "--inspect=false", "db"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 1, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "allow_method") {
		t.Fatalf("stderr missing allow_method error: %q", stderr.String())
	}
}
