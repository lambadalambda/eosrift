package server

import "testing"

func TestTunnelIDFromHost(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		host         string
		tunnelDomain string
		wantID       string
		wantOK       bool
	}{
		{"basic", "abcd1234.tunnel.eosrift.com", "tunnel.eosrift.com", "abcd1234", true},
		{"with port", "abcd1234.tunnel.eosrift.com:443", "tunnel.eosrift.com", "abcd1234", true},
		{"uppercase", "ABCD1234.TUNNEL.EOSRIFT.COM", "tunnel.eosrift.com", "abcd1234", true},
		{"tunnel apex", "tunnel.eosrift.com", "tunnel.eosrift.com", "", false},
		{"other host", "eosrift.com", "tunnel.eosrift.com", "", false},
		{"nested subdomain rejected", "a.b.tunnel.eosrift.com", "tunnel.eosrift.com", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotID, ok := tunnelIDFromHost(tc.host, tc.tunnelDomain)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (id=%q)", ok, tc.wantOK, gotID)
			}
			if gotID != tc.wantID {
				t.Fatalf("id = %q, want %q", gotID, tc.wantID)
			}
		})
	}
}
