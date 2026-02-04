package config

import (
	"os"
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
	tunnelInspect := false
	in := File{
		Version:    1,
		Authtoken:  "token-123",
		ServerAddr: "https://example.com",
		HostHeader: "rewrite",
		Inspect:    &inspect,
		Tunnels: map[string]Tunnel{
			"web": {
				Proto:       "http",
				Addr:        "127.0.0.1:3000",
				Domain:      "demo.tunnel.example.com",
				BasicAuth:   "user:pass",
				HostHeader:  "rewrite",
				Inspect:     &tunnelInspect,
				InspectAddr: "127.0.0.1:4041",
			},
		},
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
	if len(out.Tunnels) != 1 {
		t.Fatalf("tunnels len = %d, want %d", len(out.Tunnels), 1)
	}
	web, ok := out.Tunnels["web"]
	if !ok {
		t.Fatalf("tunnels missing key %q", "web")
	}
	if web.Proto != "http" || web.Addr != "127.0.0.1:3000" {
		t.Fatalf("web tunnel = %+v, want proto=http addr=127.0.0.1:3000", web)
	}
	if web.Domain != "demo.tunnel.example.com" || web.HostHeader != "rewrite" {
		t.Fatalf("web http opts = %+v, want domain/host_header set", web)
	}
	if web.BasicAuth != "user:pass" {
		t.Fatalf("web basic_auth = %q, want %q", web.BasicAuth, "user:pass")
	}
	if web.Inspect == nil || *web.Inspect != false || web.InspectAddr != "127.0.0.1:4041" {
		t.Fatalf("web inspect = %+v, want inspect=false inspect_addr=127.0.0.1:4041", web)
	}
}

func TestLoad_Tunnels(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "eosrift.yml")

	if err := os.WriteFile(path, []byte(`version: 1
authtoken: token-123
server_addr: https://example.com
tunnels:
  web:
    proto: http
    addr: 127.0.0.1:3000
    domain: demo.tunnel.example.com
    basic_auth: user:pass
  db:
    proto: tcp
    addr: 127.0.0.1:5432
    remote_port: 20001
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatalf("Load ok = false, want true")
	}

	if cfg.Authtoken != "token-123" {
		t.Fatalf("authtoken = %q, want %q", cfg.Authtoken, "token-123")
	}
	if cfg.ServerAddr != "https://example.com" {
		t.Fatalf("server_addr = %q, want %q", cfg.ServerAddr, "https://example.com")
	}

	if len(cfg.Tunnels) != 2 {
		t.Fatalf("tunnels len = %d, want %d", len(cfg.Tunnels), 2)
	}

	web := cfg.Tunnels["web"]
	if web.Proto != "http" || web.Addr != "127.0.0.1:3000" || web.Domain != "demo.tunnel.example.com" || web.BasicAuth != "user:pass" {
		t.Fatalf("web tunnel = %+v, want http tunnel fields set", web)
	}

	db := cfg.Tunnels["db"]
	if db.Proto != "tcp" || db.Addr != "127.0.0.1:5432" || db.RemotePort != 20001 {
		t.Fatalf("db tunnel = %+v, want tcp tunnel fields set", db)
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
