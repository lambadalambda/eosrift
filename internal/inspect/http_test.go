package inspect

import (
	"context"
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

	ts := httptest.NewServer(Handler(s, HandlerOptions{}))
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

func TestHandler_Replay(t *testing.T) {
	t.Parallel()

	s := NewStore(StoreConfig{MaxEntries: 10})
	e := s.Add(Entry{Method: "GET", Path: "/hello"})

	called := make(chan struct{}, 1)
	ts := httptest.NewServer(Handler(s, HandlerOptions{
		Replay: func(ctx context.Context, entry Entry) (ReplayResult, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			if entry.ID != e.ID {
				t.Fatalf("entry id = %q, want %q", entry.ID, e.ID)
			}
			return ReplayResult{StatusCode: 204}, nil
		},
	}))
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/api/requests/"+e.ID+"/replay", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body replayResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "" {
		t.Fatalf("error = %q, want empty", body.Error)
	}
	if body.StatusCode != 204 {
		t.Fatalf("status_code = %d, want %d", body.StatusCode, 204)
	}
	select {
	case <-called:
	default:
		t.Fatalf("replay not called")
	}
}
