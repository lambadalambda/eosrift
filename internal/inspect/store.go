package inspect

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Entry struct {
	ID string `json:"id"`

	StartedAt  time.Time `json:"started_at"`
	DurationMs int64     `json:"duration_ms"`

	TunnelID string `json:"tunnel_id,omitempty"`

	Method string `json:"method"`
	Path   string `json:"path"`
	Host   string `json:"host,omitempty"`

	StatusCode int `json:"status_code,omitempty"`

	BytesIn  int64 `json:"bytes_in,omitempty"`
	BytesOut int64 `json:"bytes_out,omitempty"`

	RequestHeaders  http.Header `json:"request_headers,omitempty"`
	ResponseHeaders http.Header `json:"response_headers,omitempty"`
}

type StoreConfig struct {
	MaxEntries int
}

type Store struct {
	mu sync.Mutex

	maxEntries int
	nextID     uint64
	entries    []Entry
}

func NewStore(cfg StoreConfig) *Store {
	maxEntries := cfg.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 200
	}

	return &Store{
		maxEntries: maxEntries,
	}
}

func (s *Store) Add(e Entry) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	e.ID = strconv.FormatUint(s.nextID, 10)
	if e.StartedAt.IsZero() {
		e.StartedAt = time.Now().UTC()
	}

	e = redactEntry(e)

	s.entries = append(s.entries, e)
	if len(s.entries) > s.maxEntries {
		s.entries = s.entries[len(s.entries)-s.maxEntries:]
	}

	return e
}

func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Entry, len(s.entries))
	copy(out, s.entries)

	// Return newest-first (ngrok-like).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}

func (s *Store) Get(id string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.entries {
		if s.entries[i].ID == id {
			return s.entries[i], true
		}
	}
	return Entry{}, false
}
