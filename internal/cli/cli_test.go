package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
