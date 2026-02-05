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
				Proto:                "http",
				Addr:                 "127.0.0.1:3000",
				Domain:               "demo.tunnel.example.com",
				BasicAuth:            "user:pass",
				AllowMethod:          []string{"GET"},
				AllowPath:            []string{"/healthz"},
				AllowPathPrefix:      []string{"/api/"},
				AllowCIDR:            []string{"1.2.3.0/24"},
				DenyCIDR:             []string{"1.2.3.4/32"},
				RequestHeaderAdd:     HeaderAddList{"X-Req: yes"},
				RequestHeaderRemove:  []string{"X-Remove"},
				ResponseHeaderAdd:    HeaderAddList{"X-Resp: ok"},
				ResponseHeaderRemove: []string{"X-Upstream"},
				HostHeader:           "rewrite",
				Inspect:              &tunnelInspect,
				InspectAddr:          "127.0.0.1:4041",
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
	if len(web.AllowMethod) != 1 || web.AllowMethod[0] != "GET" {
		t.Fatalf("web allow_method = %#v, want %q", web.AllowMethod, "GET")
	}
	if len(web.AllowPath) != 1 || web.AllowPath[0] != "/healthz" {
		t.Fatalf("web allow_path = %#v, want %q", web.AllowPath, "/healthz")
	}
	if len(web.AllowPathPrefix) != 1 || web.AllowPathPrefix[0] != "/api/" {
		t.Fatalf("web allow_path_prefix = %#v, want %q", web.AllowPathPrefix, "/api/")
	}
	if len(web.AllowCIDR) != 1 || web.AllowCIDR[0] != "1.2.3.0/24" {
		t.Fatalf("web allow_cidr = %#v, want %q", web.AllowCIDR, "1.2.3.0/24")
	}
	if len(web.DenyCIDR) != 1 || web.DenyCIDR[0] != "1.2.3.4/32" {
		t.Fatalf("web deny_cidr = %#v, want %q", web.DenyCIDR, "1.2.3.4/32")
	}
	if len(web.RequestHeaderAdd) != 1 || web.RequestHeaderAdd[0] != "X-Req: yes" {
		t.Fatalf("web request_header_add = %#v, want %q", web.RequestHeaderAdd, "X-Req: yes")
	}
	if len(web.RequestHeaderRemove) != 1 || web.RequestHeaderRemove[0] != "X-Remove" {
		t.Fatalf("web request_header_remove = %#v, want %q", web.RequestHeaderRemove, "X-Remove")
	}
	if len(web.ResponseHeaderAdd) != 1 || web.ResponseHeaderAdd[0] != "X-Resp: ok" {
		t.Fatalf("web response_header_add = %#v, want %q", web.ResponseHeaderAdd, "X-Resp: ok")
	}
	if len(web.ResponseHeaderRemove) != 1 || web.ResponseHeaderRemove[0] != "X-Upstream" {
		t.Fatalf("web response_header_remove = %#v, want %q", web.ResponseHeaderRemove, "X-Upstream")
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
    allow_method:
      - GET
    allow_path:
      - /healthz
    allow_path_prefix:
      - /api/
    allow_cidr:
      - 1.2.3.0/24
    request_header_add:
      - X-Req: yes
    request_header_remove:
      - X-Remove
    response_header_add:
      - X-Resp: ok
    response_header_remove:
      - X-Upstream
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
	if len(web.AllowMethod) != 1 || web.AllowMethod[0] != "GET" {
		t.Fatalf("web allow_method = %#v, want %q", web.AllowMethod, "GET")
	}
	if len(web.AllowPath) != 1 || web.AllowPath[0] != "/healthz" {
		t.Fatalf("web allow_path = %#v, want %q", web.AllowPath, "/healthz")
	}
	if len(web.AllowPathPrefix) != 1 || web.AllowPathPrefix[0] != "/api/" {
		t.Fatalf("web allow_path_prefix = %#v, want %q", web.AllowPathPrefix, "/api/")
	}
	if len(web.AllowCIDR) != 1 || web.AllowCIDR[0] != "1.2.3.0/24" {
		t.Fatalf("web allow_cidr = %#v, want %q", web.AllowCIDR, "1.2.3.0/24")
	}
	if len(web.RequestHeaderAdd) != 1 || web.RequestHeaderAdd[0] != "X-Req: yes" {
		t.Fatalf("web request_header_add = %#v, want %q", web.RequestHeaderAdd, "X-Req: yes")
	}
	if len(web.RequestHeaderRemove) != 1 || web.RequestHeaderRemove[0] != "X-Remove" {
		t.Fatalf("web request_header_remove = %#v, want %q", web.RequestHeaderRemove, "X-Remove")
	}
	if len(web.ResponseHeaderAdd) != 1 || web.ResponseHeaderAdd[0] != "X-Resp: ok" {
		t.Fatalf("web response_header_add = %#v, want %q", web.ResponseHeaderAdd, "X-Resp: ok")
	}
	if len(web.ResponseHeaderRemove) != 1 || web.ResponseHeaderRemove[0] != "X-Upstream" {
		t.Fatalf("web response_header_remove = %#v, want %q", web.ResponseHeaderRemove, "X-Upstream")
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
