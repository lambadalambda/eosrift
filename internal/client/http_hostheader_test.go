package client

import "testing"

func TestValidateHostHeaderMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in      string
		wantErr bool
	}{
		{in: "", wantErr: false},
		{in: "preserve", wantErr: false},
		{in: "rewrite", wantErr: false},
		{in: "example.com", wantErr: false},
		{in: "example.com:1234", wantErr: false},
		{in: "bad host", wantErr: true},
		{in: "bad\r\nx: y", wantErr: true},
		{in: "bad\nx: y", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			err := ValidateHostHeaderMode(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateHostHeaderMode(%q) err = nil, want non-nil", tc.in)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateHostHeaderMode(%q) err = %v, want nil", tc.in, err)
			}
		})
	}
}
