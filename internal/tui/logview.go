// Package tui provides the terminal user interface components.
package tui

import (
	"fmt"
	"strings"

	"github.com/Expert21/argus/internal/ingest"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// LogView displays the scrollable log entries.
type LogView struct {
	// viewport handles scrolling
	viewport viewport.Model

	// entries holds all log entries
	entries []ingest.LogEntry

	// width and height of the view
	width, height int

	// focused indicates if this view has focus
	focused bool

	// autoScroll follows new entries
	autoScroll bool

	// maxEntries limits memory usage
	maxEntries int

	// sourceFilter filters to a specific source (empty = show all)
	sourceFilter string

	// filteredCount tracks visible entries after filtering
	filteredCount int
}

// NewLogView creates a new log view.
func NewLogView() *LogView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle()

	return &LogView{
		viewport:     vp,
		entries:      make([]ingest.LogEntry, 0),
		autoScroll:   true,
		maxEntries:   1000,
		sourceFilter: "",
	}
}

// SetSize updates the view dimensions.
func (lv *LogView) SetSize(width, height int) {
	lv.width = width
	lv.height = height
	lv.viewport.Width = width - 4
	lv.viewport.Height = height - 4 // Account for borders + header
	lv.updateContent()
}

// SetFocused sets the focus state.
func (lv *LogView) SetFocused(focused bool) {
	lv.focused = focused
}

// SetSourceFilter sets the source filter.
func (lv *LogView) SetSourceFilter(source string) {
	lv.sourceFilter = source
	lv.updateContent()
	if lv.autoScroll {
		lv.viewport.GotoBottom()
	}
}

// AddEntry adds a new log entry.
func (lv *LogView) AddEntry(entry ingest.LogEntry) {
	lv.entries = append(lv.entries, entry)

	// Trim to max entries
	if len(lv.entries) > lv.maxEntries {
		lv.entries = lv.entries[len(lv.entries)-lv.maxEntries:]
	}

	lv.updateContent()

	// Auto-scroll to bottom if enabled
	if lv.autoScroll {
		lv.viewport.GotoBottom()
	}
}

// Clear removes all entries.
func (lv *LogView) Clear() {
	lv.entries = make([]ingest.LogEntry, 0)
	lv.updateContent()
}

// updateContent rebuilds the viewport content with filtering.
func (lv *LogView) updateContent() {
	var lines []string
	contentWidth := lv.width - 6

	lv.filteredCount = 0
	for _, entry := range lv.entries {
		// Apply source filter (matches on IngestorName, not Source)
		if lv.sourceFilter != "" && entry.IngestorName != lv.sourceFilter {
			continue
		}

		line := FormatLogEntry(entry, contentWidth)
		lines = append(lines, line)
		lv.filteredCount++
	}

	lv.viewport.SetContent(strings.Join(lines, "\n"))
}

// View renders the log view.
func (lv *LogView) View() string {
	// Header with filter info
	var headerText string
	if lv.sourceFilter == "" {
		headerText = "📜 Log Stream (All Sources)"
	} else {
		headerText = fmt.Sprintf("📜 Log Stream [%s]", lv.sourceFilter)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(headerText)

	// Scroll indicator
	scrollInfo := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(lv.scrollIndicator())

	// Entry count
	countInfo := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(fmt.Sprintf("%d entries", lv.filteredCount))

	// Build header line
	headerWidth := lipgloss.Width(header)
	scrollWidth := lipgloss.Width(scrollInfo)
	countWidth := lipgloss.Width(countInfo)
	spacing := lv.width - headerWidth - scrollWidth - countWidth - 8
	if spacing < 1 {
		spacing = 1
	}

	headerLine := header + strings.Repeat(" ", spacing) + countInfo + "  " + scrollInfo

	// Content
	var content string
	if len(lv.entries) == 0 {
		content = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true).
			Render("\n  Waiting for log entries...\n\n  Log events will appear here as they arrive.\n")
	} else if lv.filteredCount == 0 {
		content = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true).
			Render(fmt.Sprintf("\n  No entries from source: %s\n\n  Select 'All Sources' or wait for new events.\n", lv.sourceFilter))
	} else {
		content = lv.viewport.View()
	}

	// Combine header and content
	inner := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		content,
	)

	// Apply border style
	style := LogViewStyle.Width(lv.width).Height(lv.height)
	if lv.focused {
		style = LogViewFocusedStyle.Width(lv.width).Height(lv.height)
	}

	return style.Render(inner)
}

// scrollIndicator returns a string showing scroll position.
func (lv *LogView) scrollIndicator() string {
	if lv.filteredCount == 0 {
		return ""
	}

	if lv.autoScroll {
		return "↓ AUTO"
	}
	percent := int(lv.viewport.ScrollPercent() * 100)
	return fmt.Sprintf("%d%%", percent)
}

// ScrollUp scrolls the viewport up.
func (lv *LogView) ScrollUp(lines int) {
	lv.autoScroll = false
	lv.viewport.LineUp(lines)
}

// ScrollDown scrolls the viewport down.
func (lv *LogView) ScrollDown(lines int) {
	lv.viewport.LineDown(lines)
	if lv.viewport.AtBottom() {
		lv.autoScroll = true
	}
}

// PageUp scrolls up one page.
func (lv *LogView) PageUp() {
	lv.autoScroll = false
	lv.viewport.ViewUp()
}

// PageDown scrolls down one page.
func (lv *LogView) PageDown() {
	lv.viewport.ViewDown()
	if lv.viewport.AtBottom() {
		lv.autoScroll = true
	}
}

// GotoTop scrolls to the top.
func (lv *LogView) GotoTop() {
	lv.autoScroll = false
	lv.viewport.GotoTop()
}

// GotoBottom scrolls to the bottom and enables auto-scroll.
func (lv *LogView) GotoBottom() {
	lv.viewport.GotoBottom()
	lv.autoScroll = true
}

// ToggleAutoScroll toggles auto-scroll mode.
func (lv *LogView) ToggleAutoScroll() {
	lv.autoScroll = !lv.autoScroll
	if lv.autoScroll {
		lv.viewport.GotoBottom()
	}
}

// EntryCount returns the number of visible entries.
func (lv *LogView) EntryCount() int {
	return lv.filteredCount
}

// TotalEntryCount returns the total number of entries.
func (lv *LogView) TotalEntryCount() int {
	return len(lv.entries)
}

// Helper max function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
