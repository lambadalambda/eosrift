package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultPath_UsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got := DefaultPath()
	want := filepath.FromSlash("/tmp/xdg/eosrift/eosrift.yml")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	inspect := true
	in := File{
		Version:    1,
		Authtoken:  "token-123",
		ServerAddr: "https://example.com",
		HostHeader: "rewrite",
		Inspect:    &inspect,
	}

	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatalf("Load ok = false, want true")
	}
	if out.Version != in.Version {
		t.Fatalf("version = %d, want %d", out.Version, in.Version)
	}
	if out.Authtoken != in.Authtoken {
		t.Fatalf("authtoken = %q, want %q", out.Authtoken, in.Authtoken)
	}
	if out.ServerAddr != in.ServerAddr {
		t.Fatalf("server_addr = %q, want %q", out.ServerAddr, in.ServerAddr)
	}
	if out.HostHeader != in.HostHeader {
		t.Fatalf("host_header = %q, want %q", out.HostHeader, in.HostHeader)
	}
	if out.Inspect == nil || *out.Inspect != *in.Inspect {
		t.Fatalf("inspect = %v, want %v", out.Inspect, in.Inspect)
	}
}

func TestControlURLFromServerAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "wss://example.com/control",
			want: "wss://example.com/control",
		},
		{
			in:   "https://example.com",
			want: "wss://example.com/control",
		},
		{
			in:   "http://127.0.0.1:8080",
			want: "ws://127.0.0.1:8080/control",
		},
		{
			in:   "example.com:443",
			want: "wss://example.com:443/control",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := ControlURLFromServerAddr(tc.in)
			if err != nil {
				t.Fatalf("ControlURLFromServerAddr: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ControlURLFromServerAddr(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
