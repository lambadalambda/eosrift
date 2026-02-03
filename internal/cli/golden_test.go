package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_Help_Golden(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want %d (stderr=%q)", code, 0, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}

	want := readGolden(t, "help.txt")
	if stdout.String() != want {
		t.Fatalf("stdout mismatch\n--- want ---\n%s\n--- got ---\n%s", want, stdout.String())
	}
}

func TestRun_ConfigUsage_Golden(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"config"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want %d", code, 2)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}

	want := readGolden(t, "config_usage.txt")
	if stderr.String() != want {
		t.Fatalf("stderr mismatch\n--- want ---\n%s\n--- got ---\n%s", want, stderr.String())
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return string(b)
}

