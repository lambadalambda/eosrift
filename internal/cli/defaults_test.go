package cli

import (
	"testing"

	"eosrift.com/eosrift/internal/config"
)

func TestResolveServerAddrDefault(t *testing.T) {
	t.Run("prefers EOSRIFT_SERVER_ADDR", func(t *testing.T) {
		t.Setenv("EOSRIFT_SERVER_ADDR", "https://env.example.com")
		t.Setenv("EOSRIFT_CONTROL_URL", "wss://ignored.example.com/control")
		got := resolveServerAddrDefault(config.File{ServerAddr: "https://ignored-cfg.example.com"})
		if got != "https://env.example.com" {
			t.Fatalf("got = %q, want %q", got, "https://env.example.com")
		}
	})

	t.Run("falls back to EOSRIFT_CONTROL_URL", func(t *testing.T) {
		t.Setenv("EOSRIFT_SERVER_ADDR", "")
		t.Setenv("EOSRIFT_CONTROL_URL", "wss://env.example.com/control")
		got := resolveServerAddrDefault(config.File{ServerAddr: "https://ignored-cfg.example.com"})
		if got != "wss://env.example.com/control" {
			t.Fatalf("got = %q, want %q", got, "wss://env.example.com/control")
		}
	})

	t.Run("falls back to config", func(t *testing.T) {
		t.Setenv("EOSRIFT_SERVER_ADDR", "")
		t.Setenv("EOSRIFT_CONTROL_URL", "")
		got := resolveServerAddrDefault(config.File{ServerAddr: "https://cfg.example.com"})
		if got != "https://cfg.example.com" {
			t.Fatalf("got = %q, want %q", got, "https://cfg.example.com")
		}
	})

	t.Run("falls back to eosrift.com", func(t *testing.T) {
		t.Setenv("EOSRIFT_SERVER_ADDR", "")
		t.Setenv("EOSRIFT_CONTROL_URL", "")
		got := resolveServerAddrDefault(config.File{})
		if got != "https://eosrift.com" {
			t.Fatalf("got = %q, want %q", got, "https://eosrift.com")
		}
	})
}

func TestResolveAuthtokenDefault(t *testing.T) {
	t.Run("prefers EOSRIFT_AUTHTOKEN", func(t *testing.T) {
		t.Setenv("EOSRIFT_AUTHTOKEN", "tok1")
		t.Setenv("EOSRIFT_AUTH_TOKEN", "tok2")
		got := resolveAuthtokenDefault(config.File{Authtoken: "tok3"})
		if got != "tok1" {
			t.Fatalf("got = %q, want %q", got, "tok1")
		}
	})

	t.Run("falls back to EOSRIFT_AUTH_TOKEN", func(t *testing.T) {
		t.Setenv("EOSRIFT_AUTHTOKEN", "")
		t.Setenv("EOSRIFT_AUTH_TOKEN", "tok2")
		got := resolveAuthtokenDefault(config.File{Authtoken: "tok3"})
		if got != "tok2" {
			t.Fatalf("got = %q, want %q", got, "tok2")
		}
	})

	t.Run("falls back to config", func(t *testing.T) {
		t.Setenv("EOSRIFT_AUTHTOKEN", "")
		t.Setenv("EOSRIFT_AUTH_TOKEN", "")
		got := resolveAuthtokenDefault(config.File{Authtoken: "tok3"})
		if got != "tok3" {
			t.Fatalf("got = %q, want %q", got, "tok3")
		}
	})
}

func TestResolveInspectDefaults(t *testing.T) {
	t.Run("enabled defaults to true", func(t *testing.T) {
		got := resolveInspectEnabledDefault(config.File{})
		if got != true {
			t.Fatalf("got = %v, want %v", got, true)
		}
	})

	t.Run("enabled respects config", func(t *testing.T) {
		disabled := false
		got := resolveInspectEnabledDefault(config.File{Inspect: &disabled})
		if got != false {
			t.Fatalf("got = %v, want %v", got, false)
		}
	})

	t.Run("addr prefers env", func(t *testing.T) {
		t.Setenv("EOSRIFT_INSPECT_ADDR", "127.0.0.1:4999")
		got := resolveInspectAddrDefault(config.File{InspectAddr: "127.0.0.1:4040"})
		if got != "127.0.0.1:4999" {
			t.Fatalf("got = %q, want %q", got, "127.0.0.1:4999")
		}
	})

	t.Run("addr falls back to config", func(t *testing.T) {
		t.Setenv("EOSRIFT_INSPECT_ADDR", "")
		got := resolveInspectAddrDefault(config.File{InspectAddr: "127.0.0.1:4999"})
		if got != "127.0.0.1:4999" {
			t.Fatalf("got = %q, want %q", got, "127.0.0.1:4999")
		}
	})

	t.Run("addr falls back to 127.0.0.1:4040", func(t *testing.T) {
		t.Setenv("EOSRIFT_INSPECT_ADDR", "")
		got := resolveInspectAddrDefault(config.File{})
		if got != "127.0.0.1:4040" {
			t.Fatalf("got = %q, want %q", got, "127.0.0.1:4040")
		}
	})
}

func TestResolveHostHeaderDefault(t *testing.T) {
	t.Run("defaults to preserve", func(t *testing.T) {
		got := resolveHostHeaderDefault(config.File{})
		if got != "preserve" {
			t.Fatalf("got = %q, want %q", got, "preserve")
		}
	})

	t.Run("uses config value", func(t *testing.T) {
		got := resolveHostHeaderDefault(config.File{HostHeader: "rewrite"})
		if got != "rewrite" {
			t.Fatalf("got = %q, want %q", got, "rewrite")
		}
	})
}
