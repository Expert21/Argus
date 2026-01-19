// Package tui provides the terminal user interface components.
package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Expert21/argus/internal/config"
	"github.com/Expert21/argus/internal/ingest"
	"github.com/charmbracelet/lipgloss"
)

// Formatter handles log entry formatting with configurable options.
type Formatter struct {
	// TimestampFormat from config (Go time format)
	TimestampFormat string

	// HighlightRules from config
	highlightRules []highlightRule
}

// highlightRule pairs a compiled regex with a style
type highlightRule struct {
	pattern *regexp.Regexp
	style   lipgloss.Style
}

// Default patterns (used if no config rules provided)
var defaultPatterns = []struct {
	pattern string
	style   lipgloss.Style
}{
	{`(?i)\b(error|err|fail|failed|failure|denied|refused|rejected|invalid|timeout|exception)\b`, KeywordErrorStyle},
	{`(?i)\b(success|succeeded|ok|done|started|loaded|accepted|allowed|connected|established)\b`, KeywordSuccessStyle},
	{`(?i)\b(sudo|root|authentication|login|logout|session|permission|ssh|password|auth|pam)\b`, KeywordSecurityStyle},
	{`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`, KeywordIPStyle},
}

// NewFormatter creates a formatter with the given configuration.
func NewFormatter(cfg *config.Config) *Formatter {
	f := &Formatter{
		TimestampFormat: "15:04:05", // Default short format for display
	}

	// Use config timestamp format if provided
	if cfg != nil && cfg.General.TimestampFormat != "" {
		// For display, use a shorter format derived from config
		// Config might have "2006-01-02 15:04:05", we just want time portion
		f.TimestampFormat = extractTimeFormat(cfg.General.TimestampFormat)
	}

	// Build highlight rules from config
	if cfg != nil && len(cfg.Highlight) > 0 {
		for _, rule := range cfg.Highlight {
			compiled, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue // Skip invalid patterns
			}
			style := parseStyle(rule.Style)
			f.highlightRules = append(f.highlightRules, highlightRule{
				pattern: compiled,
				style:   style,
			})
		}
	}

	// Add default patterns if no config rules
	if len(f.highlightRules) == 0 {
		for _, dp := range defaultPatterns {
			compiled, _ := regexp.Compile(dp.pattern)
			f.highlightRules = append(f.highlightRules, highlightRule{
				pattern: compiled,
				style:   dp.style,
			})
		}
	}

	return f
}

// extractTimeFormat extracts just the time portion from a full timestamp format.
func extractTimeFormat(fullFormat string) string {
	// If it contains date components, try to extract just time
	// Go reference: "2006-01-02 15:04:05"
	if strings.Contains(fullFormat, "15:04:05") {
		return "15:04:05"
	}
	if strings.Contains(fullFormat, "15:04") {
		return "15:04:05"
	}
	// Otherwise return as-is (might be custom)
	return fullFormat
}

// parseStyle converts a style string like "bold red" to a lipgloss.Style.
func parseStyle(styleStr string) lipgloss.Style {
	style := lipgloss.NewStyle()
	parts := strings.Fields(strings.ToLower(styleStr))

	for _, part := range parts {
		switch part {
		case "bold":
			style = style.Bold(true)
		case "dim":
			style = style.Faint(true)
		case "italic":
			style = style.Italic(true)
		case "underline":
			style = style.Underline(true)
		case "red":
			style = style.Foreground(ColorError)
		case "green":
			style = style.Foreground(ColorSuccess)
		case "yellow":
			style = style.Foreground(ColorWarning)
		case "blue":
			style = style.Foreground(ColorInfo)
		case "magenta":
			style = style.Foreground(ColorAccent)
		case "cyan":
			style = style.Foreground(lipgloss.Color("#79c0ff"))
		case "white":
			style = style.Foreground(ColorForeground)
		case "gray", "grey":
			style = style.Foreground(ColorSecondary)
		}
	}

	return style
}

// FormatEntry formats a log entry for display.
func (f *Formatter) FormatEntry(entry ingest.LogEntry, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Timestamp
	ts := TimestampStyle.Render(entry.Timestamp.Format(f.TimestampFormat))

	// Level with color
	levelStr := LevelStyle(entry.Level.String()).Render(entry.Level.String())

	// Source name (truncated/padded)
	source := truncateOrPad(entry.Source, 12)
	sourceStr := SourceNameStyle.Render(source)

	// Message with syntax highlighting
	msgWidth := maxWidth - 40
	if msgWidth < 20 {
		msgWidth = 20
	}
	msg := truncateStr(entry.Message, msgWidth)
	msg = f.highlightMessage(msg)

	return fmt.Sprintf("%s │ %s │ %s │ %s", ts, levelStr, sourceStr, msg)
}

// highlightMessage applies configured syntax highlighting.
func (f *Formatter) highlightMessage(msg string) string {
	for _, rule := range f.highlightRules {
		msg = rule.pattern.ReplaceAllStringFunc(msg, func(match string) string {
			return rule.style.Render(match)
		})
	}
	return msg
}

// ============================================================================
// Package-level functions for backward compatibility
// ============================================================================

// Default formatter instance (used when no config is available)
var defaultFormatter = NewFormatter(nil)

// SetDefaultFormatter updates the default formatter with new config.
func SetDefaultFormatter(cfg *config.Config) {
	defaultFormatter = NewFormatter(cfg)
}

// FormatLogEntry formats a log entry using the default formatter.
func FormatLogEntry(entry ingest.LogEntry, maxWidth int) string {
	return defaultFormatter.FormatEntry(entry, maxWidth)
}

// FormatLogEntryCompact formats a log entry in compact mode.
func FormatLogEntryCompact(entry ingest.LogEntry, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	ts := TimestampStyle.Render(entry.Timestamp.Format(defaultFormatter.TimestampFormat))
	levelStr := LevelStyle(entry.Level.String()).Render(entry.Level.String())

	msgWidth := maxWidth - 20
	if msgWidth < 20 {
		msgWidth = 20
	}
	msg := truncateStr(entry.Message, msgWidth)
	msg = defaultFormatter.highlightMessage(msg)

	return fmt.Sprintf("%s %s %s", ts, levelStr, msg)
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

// ============================================================================
// StatusBar
// ============================================================================

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
	var statusIndicator string
	if sb.paused {
		statusIndicator = StatusPausedStyle.Render("⏸ PAUSED")
	} else {
		statusIndicator = StatusLiveStyle.Render("● LIVE")
	}

	message := StatusTextStyle.Render(sb.status)

	stats := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(fmt.Sprintf("Events: %d │ Sources: %d", sb.eventCount, sb.sourceCount))

	help := HelpStyle.Render("[q]uit [p]ause [c]lear [Tab]focus [?]help")

	left := fmt.Sprintf("%s  %s", statusIndicator, message)
	right := fmt.Sprintf("%s  │  %s", stats, help)

	spacing := sb.width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 0 {
		spacing = 0
	}

	bar := left + strings.Repeat(" ", spacing) + right

	return StatusBarStyle.Width(sb.width).Render(bar)
}
