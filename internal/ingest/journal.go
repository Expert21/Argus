// Package ingest provides log source ingestion capabilities.
package ingest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// JournalIngestor reads logs from systemd-journald via journalctl.
//
// GO SYNTAX LESSON #19: Struct Embedding & Composition
// ====================================================
// Go doesn't have inheritance. Instead, it uses composition.
// You can embed one struct inside another to "inherit" its fields/methods.
//
// We use sync.Mutex here to protect concurrent access to our fields.
// sync.Mutex is a mutual exclusion lock - only one goroutine can hold it at a time.
type JournalIngestor struct {
	// config holds the source configuration
	config SourceConfig

	// cmd is the running journalctl process
	cmd *exec.Cmd

	// mu protects concurrent access to mutable state
	// GO SYNTAX LESSON #20: sync.Mutex
	// ================================
	// Mutex = "mutual exclusion"
	// - mu.Lock()   -> acquire the lock (blocks if held by another goroutine)
	// - mu.Unlock() -> release the lock
	// - Always use defer mu.Unlock() right after Lock() to ensure it's released
	mu sync.Mutex

	// healthy tracks whether the ingestor is functioning
	healthy bool

	// cancel is used to stop the ingestor
	cancel context.CancelFunc
}

// NewJournalIngestor creates a new journald log ingestor.
//
// GO SYNTAX LESSON #21: Constructor Pattern
// =========================================
// Go doesn't have constructors like Python's __init__.
// Instead, we use factory functions named New<Type>.
// They return a pointer to the new instance.
//
// The & operator creates a pointer to a value.
// The * operator dereferences a pointer (gets the value it points to).
func NewJournalIngestor(config SourceConfig) *JournalIngestor {
	return &JournalIngestor{
		config:  config,
		healthy: false,
	}
}

// Name returns the human-readable name of this source.
func (j *JournalIngestor) Name() string {
	return j.config.Name
}

// Healthy returns true if the ingestor is functioning normally.
func (j *JournalIngestor) Healthy() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.healthy
}

// setHealthy safely updates the healthy status.
func (j *JournalIngestor) setHealthy(healthy bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.healthy = healthy
}

// Start begins reading logs from journalctl and sends them to the channel.
//
// GO SYNTAX LESSON #22: Goroutines
// ================================
// A goroutine is a lightweight thread managed by the Go runtime.
// Start one with: go functionName() or go func() { ... }()
//
// Goroutines are cheap (a few KB of stack) so you can have thousands.
// They communicate via channels, not shared memory (though we do use
// mutexes when necessary).
func (j *JournalIngestor) Start(ctx context.Context, entries chan<- LogEntry) error {
	// Create a cancellable context for this ingestor
	// If the parent context is cancelled OR we call j.cancel(), this stops
	ctx, j.cancel = context.WithCancel(ctx)

	// Build the journalctl command
	// -o json: Output in JSON format (much easier to parse)
	// -f: Follow mode (like tail -f)
	// --no-pager: Don't use less/more
	args := []string{"-o", "json", "-f", "--no-pager"}

	// Add any custom filters from config
	args = append(args, j.config.Filters...)

	// GO SYNTAX LESSON #23: exec.Command
	// ==================================
	// exec.Command creates a command but doesn't run it yet.
	// The first argument is the program, the rest are arguments.
	//
	// cmd.Start() starts the command asynchronously
	// cmd.Wait() blocks until it finishes
	// cmd.Run() = Start() + Wait()
	j.cmd = exec.CommandContext(ctx, "journalctl", args...)

	// Get a pipe to read stdout
	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command (non-blocking)
	if err := j.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	j.setHealthy(true)

	// GO SYNTAX LESSON #24: Goroutine for Background Processing
	// ==========================================================
	// We need to read from journalctl continuously without blocking.
	// So we spawn a goroutine to handle the reading.
	go func() {
		// Ensure we clean up when done
		defer j.setHealthy(false)
		defer stdout.Close()

		// Create a buffered reader for efficient line-by-line reading
		scanner := bufio.NewScanner(stdout)

		// Read lines until context is cancelled or stream ends
		for scanner.Scan() {
			// Check if we should stop
			select {
			case <-ctx.Done():
				return
			default:
				// Continue processing
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse the JSON line
			entry, err := j.parseJournalEntry(line)
			if err != nil {
				// Log parse errors but don't stop
				// In production, we might want to count these
				continue
			}

			// Send entry to channel (non-blocking with select)
			// GO SYNTAX LESSON #25: Select Statement
			// ======================================
			// select is like switch but for channel operations.
			// It waits until one of its cases can proceed.
			// With a default case, it becomes non-blocking.
			select {
			case entries <- entry:
				// Sent successfully
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			// Could log this error
			j.setHealthy(false)
		}
	}()

	// Wait for the command to finish (in another goroutine)
	go func() {
		j.cmd.Wait()
		j.setHealthy(false)
	}()

	return nil
}

// Stop gracefully shuts down the ingestor.
func (j *JournalIngestor) Stop() error {
	if j.cancel != nil {
		j.cancel()
	}
	if j.cmd != nil && j.cmd.Process != nil {
		// Send SIGTERM to the process
		return j.cmd.Process.Kill()
	}
	return nil
}

// journalEntry represents the JSON structure from journalctl -o json
// GO SYNTAX LESSON #26: JSON Unmarshaling
// =======================================
// To parse JSON into a struct, the field names or json tags must match.
// Use json.Unmarshal([]byte, &target) to parse.
//
// journalctl outputs fields like __REALTIME_TIMESTAMP, PRIORITY, MESSAGE, etc.
type journalEntry struct {
	RealtimeTimestamp string `json:"__REALTIME_TIMESTAMP"`
	Priority          string `json:"PRIORITY"`
	Message           string `json:"MESSAGE"`
	SyslogIdentifier  string `json:"SYSLOG_IDENTIFIER"`
	SystemdUnit       string `json:"_SYSTEMD_UNIT"`
	PID               string `json:"_PID"`
	Hostname          string `json:"_HOSTNAME"`
	Transport         string `json:"_TRANSPORT"`
}

// parseJournalEntry converts a JSON line from journalctl into a LogEntry.
func (j *JournalIngestor) parseJournalEntry(line string) (LogEntry, error) {
	var je journalEntry

	// GO SYNTAX LESSON #27: Error Handling Pattern
	// =============================================
	// Go's error handling is explicit - no hidden exceptions.
	// The pattern is:
	//   result, err := doSomething()
	//   if err != nil {
	//       return ..., fmt.Errorf("context: %w", err)
	//   }
	//
	// The %w verb wraps the error, preserving the error chain.
	// You can unwrap with errors.Unwrap() or errors.Is().
	if err := json.Unmarshal([]byte(line), &je); err != nil {
		return LogEntry{}, fmt.Errorf("failed to parse journal JSON: %w", err)
	}

	// Parse timestamp
	// journalctl outputs microseconds since epoch as a string
	ts := time.Now() // default to now if parsing fails
	if je.RealtimeTimestamp != "" {
		if usec, err := strconv.ParseInt(je.RealtimeTimestamp, 10, 64); err == nil {
			ts = time.UnixMicro(usec)
		}
	}

	// Parse priority (0-7, where 0 is emergency and 7 is debug)
	level := LevelUnknown
	if je.Priority != "" {
		if prio, err := strconv.Atoi(je.Priority); err == nil {
			level = priorityToLevel(prio)
		}
	}

	// Determine source name
	source := j.config.Name
	if je.SyslogIdentifier != "" {
		source = je.SyslogIdentifier
	}

	return LogEntry{
		Timestamp:  ts,
		Source:     source,
		SourceType: SourceJournald,
		Level:      level,
		Message:    je.Message,
		Unit:       je.SystemdUnit,
		Hostname:   je.Hostname,
		PID:        parseInt(je.PID),
		Raw:        line,
		Metadata: map[string]string{
			"transport": je.Transport,
		},
	}, nil
}

// priorityToLevel converts syslog priority (0-7) to LogLevel.
// Syslog: 0=emergency, 1=alert, 2=critical, 3=error, 4=warning, 5=notice, 6=info, 7=debug
func priorityToLevel(priority int) LogLevel {
	// We invert because syslog 0 is highest severity
	switch priority {
	case 0:
		return LevelEmergency
	case 1:
		return LevelAlert
	case 2:
		return LevelCritical
	case 3:
		return LevelError
	case 4:
		return LevelWarning
	case 5:
		return LevelNotice
	case 6:
		return LevelInfo
	case 7:
		return LevelDebug
	default:
		return LevelUnknown
	}
}

// parseInt safely parses a string to int, returning 0 on error.
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// Ensure JournalIngestor implements Ingestor interface at compile time.
// GO SYNTAX LESSON #28: Interface Compliance Check
// =================================================
// This is a common Go idiom to verify a type implements an interface.
// If JournalIngestor doesn't implement Ingestor, this line causes
// a compile error, catching the bug early.
//
// The underscore _ means "discard this value" - we don't need the variable,
// just the type check.
var _ Ingestor = (*JournalIngestor)(nil)
