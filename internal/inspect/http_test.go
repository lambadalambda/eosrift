package inspect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ListRequests(t *testing.T) {
	t.Parallel()

	s := NewStore(StoreConfig{MaxEntries: 10})
	s.Add(Entry{Method: "GET", Path: "/a"})
	s.Add(Entry{Method: "POST", Path: "/b"})

	ts := httptest.NewServer(Handler(s))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/requests")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want %q", ct, "application/json")
	}

	var body listRequestsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(body.Requests) != 2 {
		t.Fatalf("len(requests) = %d, want %d", len(body.Requests), 2)
	}
	if body.Requests[0].Path != "/b" || body.Requests[1].Path != "/a" {
		t.Fatalf("paths = %q, %q, want /b,/a", body.Requests[0].Path, body.Requests[1].Path)
	}
}
