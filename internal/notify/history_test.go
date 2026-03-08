package notify

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestHistory_NewEmpty(t *testing.T) {
	h := NewHistory(10)
	entries := h.List()
	if entries == nil {
		t.Fatal("List() returned nil, want non-nil empty slice")
	}
	if len(entries) != 0 {
		t.Errorf("List() length: got %d, want 0", len(entries))
	}
}

func TestHistory_AddAndList(t *testing.T) {
	h := NewHistory(10)

	now := time.Now()
	h.Add(Entry{SessionID: "s1", SessionName: "session-1", Type: "task_complete", Message: "done", Timestamp: now})
	h.Add(Entry{SessionID: "s2", SessionName: "session-2", Type: "permission", Message: "need approval", Timestamp: now.Add(1 * time.Second)})
	h.Add(Entry{SessionID: "s3", SessionName: "session-3", Type: "task_complete", Message: "finished", Timestamp: now.Add(2 * time.Second)})

	entries := h.List()
	if len(entries) != 3 {
		t.Fatalf("List() length: got %d, want 3", len(entries))
	}

	// Verify all entries are present (List returns newest first)
	ids := make(map[string]bool)
	for _, e := range entries {
		ids[e.SessionID] = true
	}
	for _, id := range []string{"s1", "s2", "s3"} {
		if !ids[id] {
			t.Errorf("entry with SessionID %q not found in List()", id)
		}
	}
}

func TestHistory_EvictsOldest(t *testing.T) {
	h := NewHistory(3)

	base := time.Now()
	for i := range 5 {
		h.Add(Entry{
			SessionID: fmt.Sprintf("s%d", i),
			Message:   fmt.Sprintf("msg-%d", i),
			Timestamp: base.Add(time.Duration(i) * time.Second),
		})
	}

	entries := h.List()
	if len(entries) != 3 {
		t.Fatalf("List() length: got %d, want 3", len(entries))
	}

	// Only the last 3 entries (s2, s3, s4) should remain
	ids := make(map[string]bool)
	for _, e := range entries {
		ids[e.SessionID] = true
	}
	for _, id := range []string{"s2", "s3", "s4"} {
		if !ids[id] {
			t.Errorf("expected entry %q to remain after eviction, but it was not found", id)
		}
	}
	for _, id := range []string{"s0", "s1"} {
		if ids[id] {
			t.Errorf("expected entry %q to be evicted, but it was found", id)
		}
	}
}

func TestHistory_ListSortsByTimestamp(t *testing.T) {
	h := NewHistory(10)

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Add entries out of order
	h.Add(Entry{SessionID: "middle", Timestamp: base.Add(5 * time.Second)})
	h.Add(Entry{SessionID: "oldest", Timestamp: base})
	h.Add(Entry{SessionID: "newest", Timestamp: base.Add(10 * time.Second)})
	h.Add(Entry{SessionID: "second-newest", Timestamp: base.Add(8 * time.Second)})

	entries := h.List()
	if len(entries) != 4 {
		t.Fatalf("List() length: got %d, want 4", len(entries))
	}

	// Verify descending order (newest first)
	expectedOrder := []string{"newest", "second-newest", "middle", "oldest"}
	for i, want := range expectedOrder {
		if entries[i].SessionID != want {
			t.Errorf("entries[%d].SessionID: got %q, want %q", i, entries[i].SessionID, want)
		}
	}

	// Also verify timestamps are strictly descending
	for i := 1; i < len(entries); i++ {
		if !entries[i-1].Timestamp.After(entries[i].Timestamp) {
			t.Errorf("entries[%d].Timestamp (%v) should be after entries[%d].Timestamp (%v)",
				i-1, entries[i-1].Timestamp, i, entries[i].Timestamp)
		}
	}
}

func TestHistory_ConcurrentSafety(t *testing.T) {
	h := NewHistory(100)

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 5
	entriesPerWriter := 50

	// Writers: concurrently add entries
	for w := range numWriters {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := range entriesPerWriter {
				h.Add(Entry{
					SessionID: fmt.Sprintf("w%d-e%d", writerID, i),
					Message:   fmt.Sprintf("writer %d entry %d", writerID, i),
					Timestamp: time.Now(),
				})
			}
		}(w)
	}

	// Readers: concurrently list entries
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range entriesPerWriter {
				entries := h.List()
				// Just verify List() doesn't panic and returns a valid slice
				if entries == nil {
					t.Error("List() returned nil during concurrent access")
					return
				}
			}
		}()
	}

	wg.Wait()

	// After all goroutines complete, verify history is within capacity
	entries := h.List()
	if len(entries) > 100 {
		t.Errorf("history exceeded maxSize: got %d entries, want <= 100", len(entries))
	}
}
