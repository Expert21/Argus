// Package aggregate provides the central log aggregation system.
//
// The Aggregator is the heart of Argus - it:
// 1. Manages multiple log source ingestors
// 2. Collects entries into a unified stream
// 3. Maintains a ring buffer for history
// 4. Broadcasts entries to subscribers (like the TUI)
package aggregate

import (
	"context"
	"sort"
	"sync"

	"github.com/Expert21/argus/internal/ingest"
)

// GO SYNTAX LESSON #35: Ring Buffer Data Structure
// =================================================
// A ring buffer (circular buffer) is a fixed-size buffer that wraps around.
// When it's full, new items overwrite the oldest items.
// Perfect for log viewing where we only need the last N entries.
//
// We use a slice with a write index and count.
// When full, oldest entries are overwritten.

// RingBuffer is a fixed-size circular buffer for log entries.
type RingBuffer struct {
	entries []ingest.LogEntry
	size    int // Maximum capacity
	count   int // Current number of entries
	writeAt int // Next write position
	mu      sync.RWMutex
}

// NewRingBuffer creates a ring buffer with the specified capacity.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 1000 // Default to 1000 entries
	}
	return &RingBuffer{
		entries: make([]ingest.LogEntry, size),
		size:    size,
	}
}

// Push adds an entry to the buffer, overwriting oldest if full.
func (rb *RingBuffer) Push(entry ingest.LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Write at current position
	rb.entries[rb.writeAt] = entry

	// Advance write position (wrap around)
	rb.writeAt = (rb.writeAt + 1) % rb.size

	// Track count (max is size)
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll returns all entries in chronological order.
// GO SYNTAX LESSON #36: Slice Copying
// ===================================
// When returning a slice from a function, we often need to copy it
// to prevent the caller from modifying our internal data.
// copy(dst, src) copies elements and returns number copied.
func (rb *RingBuffer) GetAll() []ingest.LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]ingest.LogEntry, rb.count)

	if rb.count < rb.size {
		// Buffer not full yet, just copy from start
		copy(result, rb.entries[:rb.count])
	} else {
		// Buffer is full, need to get entries in order
		// Oldest is at writeAt, newest is at writeAt-1
		firstPart := rb.entries[rb.writeAt:]
		secondPart := rb.entries[:rb.writeAt]
		copy(result, firstPart)
		copy(result[len(firstPart):], secondPart)
	}

	return result
}

// GetLast returns the last n entries in chronological order.
func (rb *RingBuffer) GetLast(n int) []ingest.LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n <= 0 || rb.count == 0 {
		return nil
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]ingest.LogEntry, n)

	// Calculate starting position
	// Latest entry is at (writeAt - 1 + size) % size
	// We want n entries ending at that position
	start := (rb.writeAt - n + rb.size) % rb.size

	if start+n <= rb.size {
		// Contiguous block
		copy(result, rb.entries[start:start+n])
	} else {
		// Wraps around
		firstPart := rb.entries[start:]
		secondPart := rb.entries[:n-len(firstPart)]
		copy(result, firstPart)
		copy(result[len(firstPart):], secondPart)
	}

	return result
}

// Count returns the number of entries in the buffer.
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear empties the buffer.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.count = 0
	rb.writeAt = 0
}

// Subscriber represents something that wants to receive log entries.
type Subscriber struct {
	Ch     chan ingest.LogEntry
	ID     string
	closed bool
	mu     sync.Mutex
}

// Aggregator collects logs from multiple sources and distributes them.
type Aggregator struct {
	// Sources maps source name to ingestor
	sources map[string]ingest.Ingestor

	// History is the ring buffer for recent entries
	History *RingBuffer

	// Subscribers receive new entries
	subscribers []*Subscriber

	// Internal channel for incoming entries
	entryChan chan ingest.LogEntry

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAggregator creates a new aggregator with the specified buffer size.
func NewAggregator(bufferSize int) *Aggregator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Aggregator{
		sources:     make(map[string]ingest.Ingestor),
		History:     NewRingBuffer(bufferSize),
		subscribers: make([]*Subscriber, 0),
		entryChan:   make(chan ingest.LogEntry, 1000), // Buffered channel
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the aggregation loop.
// This should be called once after creating the aggregator.
func (a *Aggregator) Start() {
	// GO SYNTAX LESSON #37: Background Processing Pattern
	// ====================================================
	// A common Go pattern is to start a goroutine that loops
	// until a context is cancelled, processing items from a channel.
	go a.aggregationLoop()
}

// aggregationLoop processes incoming entries and distributes them.
func (a *Aggregator) aggregationLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return

		case entry := <-a.entryChan:
			// Add to history
			a.History.Push(entry)

			// Broadcast to all subscribers
			a.broadcast(entry)
		}
	}
}

// broadcast sends an entry to all subscribers.
func (a *Aggregator) broadcast(entry ingest.LogEntry) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, sub := range a.subscribers {
		sub.mu.Lock()
		if !sub.closed {
			// Non-blocking send with select
			select {
			case sub.Ch <- entry:
			default:
				// Subscriber's channel is full, skip
			}
		}
		sub.mu.Unlock()
	}
}

// AddSource adds and starts a new log source.
func (a *Aggregator) AddSource(source ingest.Ingestor) error {
	a.mu.Lock()
	a.sources[source.Name()] = source
	a.mu.Unlock()

	// Start the ingestor, feeding into our entry channel
	return source.Start(a.ctx, a.entryChan)
}

// RemoveSource stops and removes a log source.
func (a *Aggregator) RemoveSource(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	source, ok := a.sources[name]
	if !ok {
		return nil // Not found, nothing to do
	}

	delete(a.sources, name)
	return source.Stop()
}

// Subscribe creates a new subscriber that receives all new entries.
func (a *Aggregator) Subscribe(id string) *Subscriber {
	sub := &Subscriber{
		Ch: make(chan ingest.LogEntry, 100),
		ID: id,
	}

	a.mu.Lock()
	a.subscribers = append(a.subscribers, sub)
	a.mu.Unlock()

	return sub
}

// Unsubscribe removes a subscriber.
func (a *Aggregator) Unsubscribe(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, sub := range a.subscribers {
		if sub.ID == id {
			sub.mu.Lock()
			sub.closed = true
			close(sub.Ch)
			sub.mu.Unlock()
			// Remove from slice
			a.subscribers = append(a.subscribers[:i], a.subscribers[i+1:]...)
			return
		}
	}
}

// GetSources returns a sorted list of source names (stable order).
func (a *Aggregator) GetSources() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, 0, len(a.sources))
	for name := range a.sources {
		names = append(names, name)
	}
	// Sort for stable order in UI
	sort.Strings(names)
	return names
}

// GetSourceHealth returns health status of all sources.
func (a *Aggregator) GetSourceHealth() map[string]bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	health := make(map[string]bool, len(a.sources))
	for name, source := range a.sources {
		health[name] = source.Healthy()
	}
	return health
}

// EntryCount returns the total number of entries in history.
func (a *Aggregator) EntryCount() int {
	return a.History.Count()
}

// Stop shuts down the aggregator and all sources.
func (a *Aggregator) Stop() {
	// Cancel context (stops aggregation loop and ingestors)
	a.cancel()

	// Stop all sources
	a.mu.Lock()
	for _, source := range a.sources {
		source.Stop()
	}
	a.sources = nil
	a.mu.Unlock()

	// Close all subscriber channels
	for _, sub := range a.subscribers {
		sub.mu.Lock()
		if !sub.closed {
			sub.closed = true
			close(sub.Ch)
		}
		sub.mu.Unlock()
	}
}
