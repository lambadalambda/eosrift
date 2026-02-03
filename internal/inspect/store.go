package inspect

import (
	"strconv"
	"sync"
)

type Entry struct {
	ID string `json:"id"`

	Method string `json:"method"`
	Path   string `json:"path"`
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

