// Package tui provides the terminal user interface components.
package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Expert21/argus/internal/ingest"
	"github.com/charmbracelet/lipgloss"
)

// LogFormatter handles the styling and formatting of log entries.
//
// GO SYNTAX LESSON #42: Syntax Highlighting with Regex
// =====================================================
// We use compiled regexes to find and highlight keywords.
// Each keyword match is wrapped with ANSI escape codes via Lipgloss.
// The order of highlighting matters - more specific patterns first.

// Regex patterns for syntax highlighting
var (
	// Error keywords
	errorPattern = regexp.MustCompile(`(?i)\b(error|err|fail|failed|failure|denied|refused|rejected|invalid|timeout|exception)\b`)

	// Success keywords
	successPattern = regexp.MustCompile(`(?i)\b(success|succeeded|ok|done|started|loaded|accepted|allowed|connected|established)\b`)

	// Security keywords
	securityPattern = regexp.MustCompile(`(?i)\b(sudo|root|authentication|login|logout|session|permission|denied|ssh|password|auth|pam)\b`)

	// IP addresses
	ipPattern = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)

	// UUIDs
	uuidPattern = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)

	// Paths
	pathPattern = regexp.MustCompile(`/[\w./\-_]+`)
)

// FormatLogEntry formats a single log entry with full styling.
func FormatLogEntry(entry ingest.LogEntry, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Timestamp
	ts := TimestampStyle.Render(entry.Timestamp.Format("15:04:05"))

	// Level with color
	levelStr := LevelStyle(entry.Level.String()).Render(entry.Level.String())

	// Source name (truncated/padded)
	source := truncateOrPad(entry.Source, 12)
	sourceStr := SourceNameStyle.Render(source)

	// Message with syntax highlighting
	msgWidth := maxWidth - 40 // Account for timestamp, level, source, spacing
	if msgWidth < 20 {
		msgWidth = 20
	}
	msg := truncateStr(entry.Message, msgWidth)
	msg = highlightMessage(msg)

	return fmt.Sprintf("%s │ %s │ %s │ %s", ts, levelStr, sourceStr, msg)
}

// FormatLogEntryCompact formats a log entry in compact mode (no source).
func FormatLogEntryCompact(entry ingest.LogEntry, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Timestamp
	ts := TimestampStyle.Render(entry.Timestamp.Format("15:04:05"))

	// Level with color
	levelStr := LevelStyle(entry.Level.String()).Render(entry.Level.String())

	// Message
	msgWidth := maxWidth - 20
	if msgWidth < 20 {
		msgWidth = 20
	}
	msg := truncateStr(entry.Message, msgWidth)
	msg = highlightMessage(msg)

	return fmt.Sprintf("%s %s %s", ts, levelStr, msg)
}

// highlightMessage applies syntax highlighting to a message.
func highlightMessage(msg string) string {
	// Apply highlighting in order of specificity

	// Highlight IPs
	msg = ipPattern.ReplaceAllStringFunc(msg, func(match string) string {
		return KeywordIPStyle.Render(match)
	})

	// Highlight errors
	msg = errorPattern.ReplaceAllStringFunc(msg, func(match string) string {
		return KeywordErrorStyle.Render(match)
	})

	// Highlight success
	msg = successPattern.ReplaceAllStringFunc(msg, func(match string) string {
		return KeywordSuccessStyle.Render(match)
	})

	// Highlight security
	msg = securityPattern.ReplaceAllStringFunc(msg, func(match string) string {
		return KeywordSecurityStyle.Render(match)
	})

	return msg
}

// truncateStr truncates a string to maxLen, adding ellipsis if needed.
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

// truncateOrPad truncates or pads a string to exactly the given length.
func truncateOrPad(s string, length int) string {
	if length <= 0 {
		return ""
	}
	if len(s) > length {
		return s[:length-1] + "…"
	}
	return s + strings.Repeat(" ", length-len(s))
}

// StatusBar renders the status bar at the bottom.
type StatusBar struct {
	status      string
	paused      bool
	eventCount  int
	sourceCount int
	width       int
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	return &StatusBar{
		status: "Starting...",
	}
}

// Update updates the status bar state.
func (sb *StatusBar) Update(status string, paused bool, eventCount, sourceCount int) {
	sb.status = status
	sb.paused = paused
	sb.eventCount = eventCount
	sb.sourceCount = sourceCount
}

// SetWidth sets the status bar width.
func (sb *StatusBar) SetWidth(width int) {
	sb.width = width
}

// View renders the status bar.
func (sb *StatusBar) View() string {
	// Left side: status indicator
	var statusIndicator string
	if sb.paused {
		statusIndicator = StatusPausedStyle.Render("⏸ PAUSED")
	} else {
		statusIndicator = StatusLiveStyle.Render("● LIVE")
	}

	// Center: message
	message := StatusTextStyle.Render(sb.status)

	// Right side: stats
	stats := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(fmt.Sprintf("Events: %d │ Sources: %d", sb.eventCount, sb.sourceCount))

	// Help text
	help := HelpStyle.Render("[q]uit [p]ause [c]lear [/]search [Tab]focus")

	// Build the bar
	left := fmt.Sprintf("%s  %s", statusIndicator, message)
	right := fmt.Sprintf("%s  │  %s", stats, help)

	// Calculate spacing
	spacing := sb.width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 0 {
		spacing = 0
	}

	bar := left + strings.Repeat(" ", spacing) + right

	return StatusBarStyle.Width(sb.width).Render(bar)
}
