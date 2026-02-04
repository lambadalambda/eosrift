package server

import (
	"strings"
	"testing"

	"eosrift.com/eosrift/internal/control"
)

func TestDecodeBaseRequest_RejectsLargeRequest(t *testing.T) {
	t.Parallel()

	small := `{"type":"http","authtoken":"ok"}`
	if _, err := decodeBaseRequest(strings.NewReader(small)); err != nil {
		t.Fatalf("small decode err = %v, want nil", err)
	}

	hugeToken := strings.Repeat("a", maxControlRequestBytes+1024)
	huge := `{"type":"http","authtoken":"` + hugeToken + `"}`
	if _, err := decodeBaseRequest(strings.NewReader(huge)); err == nil {
		t.Fatalf("huge decode err = nil, want non-nil")
	}
}

func TestParseHeaderKVList_RejectsCRLF(t *testing.T) {
	t.Parallel()

	if _, err := parseHeaderKVList("request_header_add", []control.HeaderKV{
		{Name: "X-Test", Value: "ok\r\nX-Evil: 1"},
	}); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestParseHeaderKVList_RejectsTooManyEntries(t *testing.T) {
	t.Parallel()

	values := make([]control.HeaderKV, 0, maxHeaderTransformEntries+1)
	for i := 0; i < maxHeaderTransformEntries+1; i++ {
		values = append(values, control.HeaderKV{Name: "X-Test", Value: "ok"})
	}

	if _, err := parseHeaderKVList("request_header_add", values); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestParseHeaderNameList_RejectsTooManyEntries(t *testing.T) {
	t.Parallel()

	values := make([]string, 0, maxHeaderTransformEntries+1)
	for i := 0; i < maxHeaderTransformEntries+1; i++ {
		values = append(values, "X-Test")
	}

	if _, err := parseHeaderNameList("request_header_remove", values); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}

func TestParseCIDRList_RejectsTooManyEntries(t *testing.T) {
	t.Parallel()

	values := make([]string, 0, maxCIDREntries+1)
	for i := 0; i < maxCIDREntries+1; i++ {
		values = append(values, "1.2.3.4/32")
	}

	if _, err := parseCIDRList("allow_cidr", values); err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
}
