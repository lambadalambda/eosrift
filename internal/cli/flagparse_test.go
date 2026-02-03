package cli

import (
	"flag"
	"testing"
)

func TestParseInterspersedFlags_AllowsFlagsAfterArgs(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	server := fs.String("server", "default", "server")
	inspect := fs.Bool("inspect", true, "inspect")
	inspectAddr := fs.String("inspect-addr", "127.0.0.1:4040", "inspect addr")

	if err := parseInterspersedFlags(fs, []string{"8080", "--server", "https://example.com", "--inspect=false", "--inspect-addr", "127.0.0.1:4041"}); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if *server != "https://example.com" {
		t.Fatalf("server = %q, want %q", *server, "https://example.com")
	}
	if *inspect != false {
		t.Fatalf("inspect = %v, want %v", *inspect, false)
	}
	if *inspectAddr != "127.0.0.1:4041" {
		t.Fatalf("inspect-addr = %q, want %q", *inspectAddr, "127.0.0.1:4041")
	}

	if fs.NArg() != 1 || fs.Arg(0) != "8080" {
		t.Fatalf("args = %v, want [8080]", fs.Args())
	}
}

func TestParseInterspersedFlags_BoolValueSeparateToken(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	inspect := fs.Bool("inspect", true, "inspect")

	if err := parseInterspersedFlags(fs, []string{"8080", "--inspect", "false"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if *inspect != false {
		t.Fatalf("inspect = %v, want %v", *inspect, false)
	}
	if fs.NArg() != 1 || fs.Arg(0) != "8080" {
		t.Fatalf("args = %v, want [8080]", fs.Args())
	}
}

func TestReorderInterspersedFlags_RespectsDoubleDash(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	_ = fs.String("server", "", "server")

	got := reorderInterspersedFlags(fs, []string{"8080", "--server", "https://example.com", "--", "--not-a-flag"})
	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want %d (%v)", len(got), 4, got)
	}
	if got[0] != "--server" || got[1] != "https://example.com" || got[2] != "8080" || got[3] != "--not-a-flag" {
		t.Fatalf("got = %v, want %v", got, []string{"--server", "https://example.com", "8080", "--not-a-flag"})
	}
}
