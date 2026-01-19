package aggregate

import (
	"testing"
	"time"

	"github.com/Expert21/argus/internal/ingest"
)

// TestRingBufferBasic tests basic ring buffer operations.
func TestRingBufferBasic(t *testing.T) {
	rb := NewRingBuffer(5)

	if rb.Count() != 0 {
		t.Errorf("Count() = %d, want 0", rb.Count())
	}

	// Add entries
	for i := 0; i < 3; i++ {
		rb.Push(ingest.LogEntry{
			Message: "entry",
			Level:   ingest.LevelInfo,
		})
	}

	if rb.Count() != 3 {
		t.Errorf("Count() = %d, want 3", rb.Count())
	}
}

// TestRingBufferOverflow tests that oldest entries are overwritten.
func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add more entries than capacity
	for i := 0; i < 5; i++ {
		rb.Push(ingest.LogEntry{
			Message: "entry",
			PID:     i,
		})
	}

	// Should only have 3 entries
	if rb.Count() != 3 {
		t.Errorf("Count() = %d, want 3", rb.Count())
	}

	// Should have entries 2, 3, 4 (oldest 0, 1 overwritten)
	entries := rb.GetAll()
	if len(entries) != 3 {
		t.Fatalf("GetAll() returned %d entries, want 3", len(entries))
	}
	if entries[0].PID != 2 {
		t.Errorf("entries[0].PID = %d, want 2", entries[0].PID)
	}
	if entries[2].PID != 4 {
		t.Errorf("entries[2].PID = %d, want 4", entries[2].PID)
	}
}

// TestRingBufferGetLast tests getting the last N entries.
func TestRingBufferGetLast(t *testing.T) {
	rb := NewRingBuffer(10)

	for i := 0; i < 7; i++ {
		rb.Push(ingest.LogEntry{PID: i})
	}

	last3 := rb.GetLast(3)
	if len(last3) != 3 {
		t.Fatalf("GetLast(3) returned %d entries, want 3", len(last3))
	}
	if last3[0].PID != 4 {
		t.Errorf("last3[0].PID = %d, want 4", last3[0].PID)
	}
	if last3[2].PID != 6 {
		t.Errorf("last3[2].PID = %d, want 6", last3[2].PID)
	}
}

// TestRingBufferClear tests clearing the buffer.
func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer(5)

	for i := 0; i < 3; i++ {
		rb.Push(ingest.LogEntry{Message: "test"})
	}

	rb.Clear()

	if rb.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", rb.Count())
	}
}

// TestAggregatorStartStop tests the aggregator lifecycle.
func TestAggregatorStartStop(t *testing.T) {
	agg := NewAggregator(100)
	agg.Start()

	// Should be able to get sources (empty)
	sources := agg.GetSources()
	if len(sources) != 0 {
		t.Errorf("GetSources() = %d sources, want 0", len(sources))
	}

	agg.Stop()
}

// TestAggregatorSubscribe tests subscription mechanism.
func TestAggregatorSubscribe(t *testing.T) {
	agg := NewAggregator(100)
	agg.Start()

	sub := agg.Subscribe("test")
	if sub == nil {
		t.Fatal("Subscribe() returned nil")
	}
	if sub.ID != "test" {
		t.Errorf("Subscriber ID = %q, want %q", sub.ID, "test")
	}

	agg.Unsubscribe("test")
	agg.Stop()
}

// TestAggregatorEntryBroadcast tests that entries are broadcast to subscribers.
func TestAggregatorEntryBroadcast(t *testing.T) {
	agg := NewAggregator(100)
	agg.Start()

	sub := agg.Subscribe("test")

	// Manually push an entry to the history and broadcast
	entry := ingest.LogEntry{
		Message:   "test entry",
		Level:     ingest.LevelInfo,
		Timestamp: time.Now(),
	}
	agg.History.Push(entry)

	// Verify entry is in history
	if agg.EntryCount() != 1 {
		t.Errorf("EntryCount() = %d, want 1", agg.EntryCount())
	}

	agg.Unsubscribe("test")
	agg.Stop()

	// Verify subscriber channel is closed
	_, ok := <-sub.Ch
	if ok {
		t.Error("Subscriber channel should be closed after unsubscribe")
	}
}
