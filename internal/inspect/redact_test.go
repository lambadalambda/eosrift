package inspect

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRedaction_RedactsSensitiveHeadersAndQueryParams(t *testing.T) {
	t.Parallel()

	s := NewStore(StoreConfig{MaxEntries: 10})

	e := s.Add(Entry{
		Method: "GET",
		Path:   "/hello?token=abc123&x=1",
		RequestHeaders: http.Header{
			"Authorization": []string{"Bearer secret"},
			"Cookie":        []string{"session=abc"},
			"X-Api-Key":     []string{"k"},
		},
		ResponseHeaders: http.Header{
			"Set-Cookie": []string{"session=def"},
		},
	})

	got, ok := s.Get(e.ID)
	if !ok {
		t.Fatalf("Get ok = false, want true")
	}

	if got.RequestHeaders.Get("Authorization") != RedactedValue {
		t.Fatalf("authorization = %q, want %q", got.RequestHeaders.Get("Authorization"), RedactedValue)
	}
	if got.RequestHeaders.Get("Cookie") != RedactedValue {
		t.Fatalf("cookie = %q, want %q", got.RequestHeaders.Get("Cookie"), RedactedValue)
	}
	if got.RequestHeaders.Get("X-Api-Key") != RedactedValue {
		t.Fatalf("x-api-key = %q, want %q", got.RequestHeaders.Get("X-Api-Key"), RedactedValue)
	}
	if got.ResponseHeaders.Get("Set-Cookie") != RedactedValue {
		t.Fatalf("set-cookie = %q, want %q", got.ResponseHeaders.Get("Set-Cookie"), RedactedValue)
	}

	u, err := url.ParseRequestURI(got.Path)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	q := u.Query()
	if q.Get("token") != RedactedValue {
		t.Fatalf("token = %q, want %q", q.Get("token"), RedactedValue)
	}
	if q.Get("x") != "1" {
		t.Fatalf("x = %q, want %q", q.Get("x"), "1")
	}
}
