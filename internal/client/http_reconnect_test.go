package client

import (
	"reflect"
	"testing"

	"eosrift.com/eosrift/internal/control"
)

func TestHTTPTunnel_ControlRequestForReconnect(t *testing.T) {
	t.Parallel()

	tun := &HTTPTunnel{
		URL:       "https://abcd1234.tunnel.eosrift.test",
		authtoken: "tok-123",
		basicAuth: "user:pass",
		allowCIDRs: []string{
			"1.2.3.4/32",
		},
		denyCIDRs: []string{
			"10.0.0.0/8",
		},
		requestHeaderAdd: []HeaderKV{
			{Name: "X-Req", Value: "yes"},
		},
		requestHeaderRemove: []string{"X-Remove"},
		responseHeaderAdd: []HeaderKV{
			{Name: "X-Resp", Value: "ok"},
		},
		responseHeaderRemove: []string{"X-Upstream"},
	}

	got := tun.controlRequestForReconnect()
	want := control.CreateHTTPTunnelRequest{
		Type:      "http",
		Authtoken: "tok-123",
		Domain:    "abcd1234.tunnel.eosrift.test",
		BasicAuth: "user:pass",
		AllowCIDR: []string{"1.2.3.4/32"},
		DenyCIDR:  []string{"10.0.0.0/8"},
		RequestHeaderAdd: []control.HeaderKV{
			{Name: "X-Req", Value: "yes"},
		},
		RequestHeaderRemove: []string{"X-Remove"},
		ResponseHeaderAdd: []control.HeaderKV{
			{Name: "X-Resp", Value: "ok"},
		},
		ResponseHeaderRemove: []string{"X-Upstream"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request mismatch\n got: %+v\nwant: %+v", got, want)
	}
}
