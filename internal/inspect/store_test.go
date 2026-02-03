package inspect

import "testing"

func TestStore_AddAndList_RetainsNewestFirst(t *testing.T) {
	t.Parallel()

	s := NewStore(StoreConfig{MaxEntries: 3})

	s.Add(Entry{Method: "GET", Path: "/1"})
	s.Add(Entry{Method: "GET", Path: "/2"})
	s.Add(Entry{Method: "GET", Path: "/3"})
	s.Add(Entry{Method: "GET", Path: "/4"})

	got := s.List()
	if len(got) != 3 {
		t.Fatalf("len(List) = %d, want %d", len(got), 3)
	}

	if got[0].Path != "/4" || got[1].Path != "/3" || got[2].Path != "/2" {
		t.Fatalf("List paths = %q, %q, %q, want /4,/3,/2", got[0].Path, got[1].Path, got[2].Path)
	}
	if got[0].ID == "" || got[1].ID == "" || got[2].ID == "" {
		t.Fatalf("expected IDs to be assigned")
	}
}
