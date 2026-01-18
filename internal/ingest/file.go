// Package ingest provides log source ingestion capabilities.
package ingest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileIngestor watches a single log file for new lines.
//
// GO SYNTAX LESSON #29: File I/O in Go
// ====================================
// Go's os package provides file operations:
// - os.Open(path) - open for reading
// - os.Create(path) - create/truncate for writing
// - os.OpenFile(path, flags, perm) - full control
//
// Files implement io.Reader and io.Writer interfaces.
// Always close files with defer file.Close().
type FileIngestor struct {
	config  SourceConfig
	watcher *fsnotify.Watcher
	file    *os.File
	mu      sync.Mutex
	healthy bool
	cancel  context.CancelFunc
	offset  int64 // Current read position in file
}

// NewFileIngestor creates a new file-watching ingestor.
func NewFileIngestor(config SourceConfig) *FileIngestor {
	return &FileIngestor{
		config:  config,
		healthy: false,
	}
}

// Name returns the human-readable name of this source.
func (f *FileIngestor) Name() string {
	return f.config.Name
}

// Healthy returns true if the ingestor is functioning normally.
func (f *FileIngestor) Healthy() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.healthy
}

func (f *FileIngestor) setHealthy(healthy bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.healthy = healthy
}

// Start begins watching the file and sends new lines to the channel.
func (f *FileIngestor) Start(ctx context.Context, entries chan<- LogEntry) error {
	ctx, f.cancel = context.WithCancel(ctx)

	// Verify the file exists
	if _, err := os.Stat(f.config.Path); err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	// Create fsnotify watcher
	// GO SYNTAX LESSON #30: fsnotify
	// ==============================
	// fsnotify wraps OS-specific file watching:
	// - Linux: inotify
	// - macOS: FSEvents
	// - Windows: ReadDirectoryChangesW
	//
	// Events: Create, Write, Remove, Rename, Chmod
	var err error
	f.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Open the file
	f.file, err = os.Open(f.config.Path)
	if err != nil {
		f.watcher.Close()
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Seek to end of file (we only want new lines)
	// GO SYNTAX LESSON #31: File Seeking
	// ==================================
	// Seek(offset, whence) moves the read/write position:
	// - io.SeekStart (0) - relative to start of file
	// - io.SeekCurrent (1) - relative to current position
	// - io.SeekEnd (2) - relative to end of file
	f.offset, err = f.file.Seek(0, io.SeekEnd)
	if err != nil {
		f.file.Close()
		f.watcher.Close()
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	// Add the file to the watcher
	if err := f.watcher.Add(f.config.Path); err != nil {
		f.file.Close()
		f.watcher.Close()
		return fmt.Errorf("failed to watch file: %w", err)
	}

	f.setHealthy(true)

	// Start the file watcher goroutine
	go f.watchLoop(ctx, entries)

	return nil
}

// watchLoop handles fsnotify events and reads new lines.
func (f *FileIngestor) watchLoop(ctx context.Context, entries chan<- LogEntry) {
	defer f.setHealthy(false)
	defer f.file.Close()
	defer f.watcher.Close()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}

			// We only care about write events
			// GO SYNTAX LESSON #32: Bitwise Operations
			// ========================================
			// fsnotify uses bitmasks for event types.
			// event.Op & fsnotify.Write checks if the Write bit is set.
			//
			// Bitwise operators:
			// & (AND), | (OR), ^ (XOR), &^ (AND NOT)
			if event.Op&fsnotify.Write == fsnotify.Write {
				f.readNewLines(entries)
			}

			// Handle log rotation (file was renamed/removed and recreated)
			if event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				f.handleRotation(ctx, entries)
			}

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			// In production, we might emit an error event
			_ = err // TODO: proper error handling
		}
	}
}

// readNewLines reads any new content from the file since last read.
func (f *FileIngestor) readNewLines(entries chan<- LogEntry) {
	// Get current file size
	info, err := f.file.Stat()
	if err != nil {
		return
	}

	// If file was truncated (size < offset), reset to beginning
	if info.Size() < f.offset {
		f.offset = 0
		f.file.Seek(0, io.SeekStart)
	}

	// Read from current offset
	f.file.Seek(f.offset, io.SeekStart)
	reader := bufio.NewReader(f.file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Real error
				f.setHealthy(false)
			}
			break
		}

		// Update offset
		f.offset += int64(len(line))

		// Trim newline and skip empty lines
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}

		// Parse the line and send it
		entry := f.parseLine(line)
		select {
		case entries <- entry:
		default:
			// Channel full, skip this entry
		}
	}
}

// handleRotation handles log file rotation.
func (f *FileIngestor) handleRotation(ctx context.Context, entries chan<- LogEntry) {
	// Close current file
	f.file.Close()

	// Wait a bit for the new file to be created
	// GO SYNTAX LESSON #33: time.Sleep and time.After
	// ================================================
	// time.Sleep blocks the current goroutine.
	// time.After returns a channel that receives after duration.
	// time.Tick returns a channel that receives periodically.
	time.Sleep(100 * time.Millisecond)

	// Try to reopen the file
	var err error
	for i := 0; i < 10; i++ {
		f.file, err = os.Open(f.config.Path)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		f.setHealthy(false)
		return
	}

	// Reset offset to start of new file
	f.offset = 0
}

// parseLine attempts to parse a log line into a LogEntry.
// It tries common log formats (syslog, timestamp-based, etc.)
func (f *FileIngestor) parseLine(line string) LogEntry {
	entry := LogEntry{
		Source:     f.config.Name,
		SourceType: SourceFile,
		Raw:        line,
		Message:    line, // Default: whole line is the message
		Timestamp:  time.Now(),
		Level:      LevelUnknown,
		Metadata:   make(map[string]string),
	}

	// Try to parse syslog format
	// Example: Jan 18 15:04:05 hostname process[pid]: message
	if parsed := parseSyslogLine(line); parsed != nil {
		entry.Timestamp = parsed.timestamp
		entry.Message = parsed.message
		entry.Hostname = parsed.hostname
		entry.Metadata["process"] = parsed.process
	}

	// Detect log level from content
	entry.Level = detectLevel(line)

	return entry
}

// Stop gracefully shuts down the ingestor.
func (f *FileIngestor) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	return nil
}

// syslogParsed holds the result of parsing a syslog line
type syslogParsed struct {
	timestamp time.Time
	hostname  string
	process   string
	message   string
}

// syslogRegex matches standard syslog format
// GO SYNTAX LESSON #34: Compiled Regex
// ====================================
// regexp.MustCompile panics if the pattern is invalid.
// Use it for patterns known at compile time.
// regexp.Compile returns an error instead of panicking.
//
// Store compiled regexes at package level to avoid recompiling.
var syslogRegex = regexp.MustCompile(
	`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+?)(?:\[\d+\])?:\s*(.*)$`,
)

// parseSyslogLine attempts to parse a syslog-formatted line.
func parseSyslogLine(line string) *syslogParsed {
	matches := syslogRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	// Parse timestamp (add current year since syslog doesn't include it)
	ts, err := time.Parse("Jan 2 15:04:05", matches[1])
	if err != nil {
		ts = time.Now()
	} else {
		// Set year to current year
		ts = ts.AddDate(time.Now().Year(), 0, 0)
	}

	return &syslogParsed{
		timestamp: ts,
		hostname:  matches[2],
		process:   matches[3],
		message:   matches[4],
	}
}

// detectLevel looks for level keywords in the log line.
func detectLevel(line string) LogLevel {
	upper := strings.ToUpper(line)

	// Check in order of severity (most severe first)
	switch {
	case strings.Contains(upper, "EMERG") || strings.Contains(upper, "EMERGENCY"):
		return LevelEmergency
	case strings.Contains(upper, "ALERT"):
		return LevelAlert
	case strings.Contains(upper, "CRIT") || strings.Contains(upper, "CRITICAL"):
		return LevelCritical
	case strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR"):
		return LevelError
	case strings.Contains(upper, "WARN") || strings.Contains(upper, "WARNING"):
		return LevelWarning
	case strings.Contains(upper, "NOTICE"):
		return LevelNotice
	case strings.Contains(upper, "INFO"):
		return LevelInfo
	case strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE"):
		return LevelDebug
	default:
		return LevelUnknown
	}
}

// Ensure FileIngestor implements Ingestor
var _ Ingestor = (*FileIngestor)(nil)
