// Package ingest provides log source ingestion capabilities.
//
// This package defines the core interfaces and types for reading logs from
// various sources (journald, files, directories) and emitting them as a
// unified stream of LogEntry values.
//
// GO SYNTAX LESSON #10: Packages & Internal Directory
// ====================================================
// The "internal" directory is special in Go:
// - Code in internal/ can only be imported by code in the parent directory
// - This prevents external packages from depending on your internal code
// - It's Go's way of marking "private" packages
//
// Package naming convention:
// - Package name matches the directory name (ingest/)
// - Short, lowercase, no underscores
// - Describes what the package DOES, not what it CONTAINS
package ingest

import (
	"context" // For cancellation and timeouts
	"time"    // For timestamps
)

// GO SYNTAX LESSON #11: Custom Types & iota
// =========================================
// Go lets you create new types based on existing ones.
// This is how you create type-safe enums in Go.
//
// iota is a special constant that starts at 0 and increments by 1
// for each constant in the block.

// LogLevel represents the severity of a log entry.
// Maps to syslog priorities (0=emergency, 7=debug)
type LogLevel int

// Log level constants using iota for auto-incrementing values
const (
	// LevelUnknown is used when the level cannot be determined
	LevelUnknown LogLevel = iota // 0
	// LevelDebug is for detailed debugging information
	LevelDebug // 1
	// LevelInfo is for general informational messages
	LevelInfo // 2
	// LevelNotice is for normal but significant conditions
	LevelNotice // 3
	// LevelWarning is for warning conditions
	LevelWarning // 4
	// LevelError is for error conditions
	LevelError // 5
	// LevelCritical is for critical conditions
	LevelCritical // 6
	// LevelAlert is for action must be taken immediately
	LevelAlert // 7
	// LevelEmergency is for system is unusable
	LevelEmergency // 8
)

// String returns a human-readable name for the log level.
//
// GO SYNTAX LESSON #12: The Stringer Interface
// =============================================
// Any type that has a String() string method automatically
// implements the fmt.Stringer interface. This means when you
// print the value with fmt.Printf("%s", level), Go will call
// this method automatically.
func (l LogLevel) String() string {
	// GO SYNTAX LESSON #13: Switch Without Expression
	// ================================================
	// Unlike other languages, Go switches don't fall through by default.
	// Each case is independent. Use "fallthrough" keyword if you need it.
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelNotice:
		return "NOTICE"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRIT"
	case LevelAlert:
		return "ALERT"
	case LevelEmergency:
		return "EMERG"
	default:
		return "UNKNOWN"
	}
}

// SourceType identifies the kind of log source
type SourceType int

const (
	// SourceJournald reads from systemd journal
	SourceJournald SourceType = iota
	// SourceFile watches a single log file
	SourceFile
	// SourceDirectory watches all log files in a directory
	SourceDirectory
)

func (s SourceType) String() string {
	switch s {
	case SourceJournald:
		return "journald"
	case SourceFile:
		return "file"
	case SourceDirectory:
		return "directory"
	default:
		return "unknown"
	}
}

// LogEntry represents a single log event from any source.
//
// GO SYNTAX LESSON #14: Struct Tags
// =================================
// The `json:"..."` after each field is a "struct tag".
// These are metadata that can be read at runtime using reflection.
// The json package uses these to know how to serialize/deserialize.
//
// Common tags:
// - json:"name"         -> Use "name" in JSON
// - json:"name,omitempty" -> Omit if zero value
// - json:"-"            -> Skip this field entirely
type LogEntry struct {
	// Timestamp when the log entry was created
	Timestamp time.Time `json:"timestamp"`

	// Source identifies where this entry came from (e.g., "journald", "auth.log")
	Source string `json:"source"`

	// SourceType indicates the type of source (journald, file, etc.)
	SourceType SourceType `json:"source_type"`

	// Level is the severity of the log entry
	Level LogLevel `json:"level"`

	// Message is the main log content
	Message string `json:"message"`

	// Unit is the systemd unit name (for journald entries)
	Unit string `json:"unit,omitempty"`

	// PID is the process ID if available
	PID int `json:"pid,omitempty"`

	// Hostname is the machine name
	Hostname string `json:"hostname,omitempty"`

	// Raw is the original unparsed line (useful for debugging)
	Raw string `json:"raw,omitempty"`

	// Metadata holds any extra fields from the source
	// GO SYNTAX LESSON #15: Maps
	// ==========================
	// map[KeyType]ValueType is Go's dictionary/hashmap.
	// - Make with make(map[string]string) or literal map[string]string{}
	// - Access: value := m["key"]
	// - Check existence: value, ok := m["key"]
	// - Delete: delete(m, "key")
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SourceConfig holds the configuration for a log source.
type SourceConfig struct {
	// Name is a human-readable identifier
	Name string `yaml:"name" json:"name"`

	// Type is the source type (journald, file, directory)
	Type SourceType `yaml:"type" json:"type"`

	// Path is the file/directory path (not used for journald)
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// Enabled controls whether this source is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Filters are optional journalctl filters (e.g., "-u nginx")
	Filters []string `yaml:"filters,omitempty" json:"filters,omitempty"`

	// GlobPattern is used for directory sources (e.g., "*.log")
	GlobPattern string `yaml:"glob,omitempty" json:"glob,omitempty"`
}

// GO SYNTAX LESSON #16: Interfaces
// ================================
// An interface defines a set of method signatures.
// Any type that implements ALL methods automatically satisfies the interface.
// This is "duck typing" at compile time - no explicit "implements" keyword.
//
// Interfaces are typically small (1-3 methods).
// The io.Reader interface is just: Read(p []byte) (n int, err error)

// Ingestor is the interface that all log sources must implement.
type Ingestor interface {
	// Start begins reading logs and sends them to the provided channel.
	// It blocks until the context is cancelled or an error occurs.
	// The channel should be closed by the caller after Start returns.
	//
	// GO SYNTAX LESSON #17: context.Context
	// =====================================
	// Context is Go's way of handling cancellation, deadlines, and
	// request-scoped values. Almost every function that does I/O
	// should take a context as its first parameter.
	//
	// Common patterns:
	// - context.Background() - root context, never cancelled
	// - context.WithCancel(parent) - can be cancelled manually
	// - context.WithTimeout(parent, duration) - auto-cancels after timeout
	//
	// Check for cancellation: if ctx.Err() != nil { return }
	// Or use select with ctx.Done()
	Start(ctx context.Context, entries chan<- LogEntry) error

	// Stop gracefully shuts down the ingestor.
	// It should return quickly and not block.
	Stop() error

	// Name returns the human-readable name of this source.
	Name() string

	// Healthy returns true if the source is functioning normally.
	Healthy() bool
}

// GO SYNTAX LESSON #18: Channels
// ==============================
// Channels are Go's primary mechanism for goroutine communication.
// Think of them as typed, thread-safe queues.
//
// chan<- T  = send-only channel (can only write to it)
// <-chan T  = receive-only channel (can only read from it)
// chan T    = bidirectional channel (read and write)
//
// Operations:
// - ch <- value     // send value to channel
// - value := <-ch   // receive value from channel
// - close(ch)       // close channel (signals no more values)
//
// The entries chan<- LogEntry in Start() means:
// "This function can only SEND entries, not receive them"
// This is a compile-time guarantee that Start won't read from the channel.
