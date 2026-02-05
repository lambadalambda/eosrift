package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_HTTP_DomainAndSubdomain_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", path,
		"http",
		"3000",
		"--domain", "demo.tunnel.eosrift.com",
		"--subdomain", "demo",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "only one of --subdomain or --domain") {
		t.Fatalf("stderr missing conflict error: %q", stderr.String())
	}
}

func TestRun_HTTP_InvalidBasicAuth_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", path,
		"http",
		"3000",
		"--basic-auth", "userpass",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--basic-auth must be in the form user:pass") {
		t.Fatalf("stderr missing basic-auth error: %q", stderr.String())
	}
}

func TestRun_HTTP_InvalidAllowCIDR_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", path,
		"http",
		"3000",
		"--allow-cidr", "nope",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid allow_cidr") {
		t.Fatalf("stderr missing allow_cidr error: %q", stderr.String())
	}
}

func TestRun_HTTP_InvalidAllowMethod_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", path,
		"http",
		"3000",
		"--allow-method", "G ET",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid allow_method") {
		t.Fatalf("stderr missing allow_method error: %q", stderr.String())
	}
}

func TestRun_HTTP_InvalidAllowPath_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", path,
		"http",
		"3000",
		"--allow-path", "healthz",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid allow_path") {
		t.Fatalf("stderr missing allow_path error: %q", stderr.String())
	}
}

func TestRun_HTTP_InvalidUpstream_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "http", "abc"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid upstream addr") {
		t.Fatalf("stderr missing upstream error: %q", stderr.String())
	}
}

func TestRun_TCP_NoArgs_IsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", path, "tcp"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 2, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage: eosrift tcp") {
		t.Fatalf("stderr missing usage: %q", stderr.String())
	}
}
