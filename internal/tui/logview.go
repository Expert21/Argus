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

	// filteredEntries holds entries after filtering (for selection indexing)
	filteredEntries []ingest.LogEntry

	// selectedIndex is the currently selected entry in filteredEntries
	selectedIndex int

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
		viewport:        vp,
		entries:         make([]ingest.LogEntry, 0),
		filteredEntries: make([]ingest.LogEntry, 0),
		selectedIndex:   0,
		autoScroll:      true,
		maxEntries:      1000,
		sourceFilter:    "",
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

	// Auto-select newest entry if auto-scroll enabled
	if lv.autoScroll {
		lv.selectedIndex = lv.filteredCount // Will be clamped in updateContent
	}

	lv.updateContent()
}

// Clear removes all entries.
func (lv *LogView) Clear() {
	lv.entries = make([]ingest.LogEntry, 0)
	lv.updateContent()
}

// updateContent rebuilds the viewport content with filtering.
func (lv *LogView) updateContent() {
	lv.filteredEntries = make([]ingest.LogEntry, 0)
	contentWidth := lv.width - 8 // Account for borders and scrollbar

	// Build filtered entries list
	for _, entry := range lv.entries {
		// Apply source filter (matches on IngestorName, not Source)
		if lv.sourceFilter != "" && entry.IngestorName != lv.sourceFilter {
			continue
		}
		lv.filteredEntries = append(lv.filteredEntries, entry)
	}
	lv.filteredCount = len(lv.filteredEntries)

	// Clamp selected index
	if lv.selectedIndex >= lv.filteredCount {
		lv.selectedIndex = lv.filteredCount - 1
	}
	if lv.selectedIndex < 0 {
		lv.selectedIndex = 0
	}

	// Build display lines
	var lines []string
	for i, entry := range lv.filteredEntries {
		line := lv.formatEntryCompact(entry, contentWidth)
		if i == lv.selectedIndex {
			// Highlight selected entry
			line = LogEntrySelectedStyle.Width(contentWidth).Render(line)
		}
		lines = append(lines, line)
	}

	lv.viewport.SetContent(strings.Join(lines, "\n"))

	// Ensure selected entry is visible
	lv.ensureSelectedVisible()
}

// formatEntryCompact formats an entry for the compact log list view.
func (lv *LogView) formatEntryCompact(entry ingest.LogEntry, maxWidth int) string {
	ts := TimestampStyle.Render(entry.Timestamp.Format("15:04:05"))
	levelStr := LevelStyle(entry.Level.String()).Render(entry.Level.String())
	source := truncateOrPad(entry.Source, 12)
	sourceStr := SourceNameStyle.Render(source)

	// Calculate remaining width for message
	msgWidth := maxWidth - 40
	if msgWidth < 20 {
		msgWidth = 20
	}
	msg := truncateStr(entry.Message, msgWidth)

	return fmt.Sprintf("%s  %s  %s  %s", ts, levelStr, sourceStr, msg)
}

// ensureSelectedVisible scrolls viewport to keep selection visible.
func (lv *LogView) ensureSelectedVisible() {
	if lv.filteredCount == 0 {
		return
	}

	visibleLines := lv.viewport.Height
	currentTop := lv.viewport.YOffset

	// If selected is above visible area, scroll up
	if lv.selectedIndex < currentTop {
		lv.viewport.SetYOffset(lv.selectedIndex)
	}
	// If selected is below visible area, scroll down
	if lv.selectedIndex >= currentTop+visibleLines {
		lv.viewport.SetYOffset(lv.selectedIndex - visibleLines + 1)
	}
}

// View renders the log view.
func (lv *LogView) View() string {
	// Hide if width is too small
	if lv.width <= 0 {
		return ""
	}

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

// SelectUp moves selection up by one entry.
func (lv *LogView) SelectUp() {
	if lv.selectedIndex > 0 {
		lv.selectedIndex--
		lv.autoScroll = false
		lv.updateContent()
	}
}

// SelectDown moves selection down by one entry.
func (lv *LogView) SelectDown() {
	if lv.selectedIndex < lv.filteredCount-1 {
		lv.selectedIndex++
		lv.updateContent()
	}
	if lv.selectedIndex == lv.filteredCount-1 {
		lv.autoScroll = true
	}
}

// PageUp moves selection up by one page.
func (lv *LogView) PageUp() {
	lv.autoScroll = false
	pageSize := lv.viewport.Height
	lv.selectedIndex -= pageSize
	if lv.selectedIndex < 0 {
		lv.selectedIndex = 0
	}
	lv.updateContent()
}

// PageDown moves selection down by one page.
func (lv *LogView) PageDown() {
	pageSize := lv.viewport.Height
	lv.selectedIndex += pageSize
	if lv.selectedIndex >= lv.filteredCount {
		lv.selectedIndex = lv.filteredCount - 1
	}
	if lv.selectedIndex == lv.filteredCount-1 {
		lv.autoScroll = true
	}
	lv.updateContent()
}

// GotoTop moves selection to the first entry.
func (lv *LogView) GotoTop() {
	lv.autoScroll = false
	lv.selectedIndex = 0
	lv.updateContent()
}

// GotoBottom moves selection to the last entry and enables auto-scroll.
func (lv *LogView) GotoBottom() {
	if lv.filteredCount > 0 {
		lv.selectedIndex = lv.filteredCount - 1
	}
	lv.autoScroll = true
	lv.updateContent()
}

// ToggleAutoScroll toggles auto-scroll mode.
func (lv *LogView) ToggleAutoScroll() {
	lv.autoScroll = !lv.autoScroll
	if lv.autoScroll && lv.filteredCount > 0 {
		lv.selectedIndex = lv.filteredCount - 1
		lv.updateContent()
	}
}

// GetSelectedEntry returns the currently selected log entry, or nil if none.
func (lv *LogView) GetSelectedEntry() *ingest.LogEntry {
	if lv.filteredCount == 0 || lv.selectedIndex < 0 || lv.selectedIndex >= lv.filteredCount {
		return nil
	}
	return &lv.filteredEntries[lv.selectedIndex]
}

// renderScrollbar renders a vertical scrollbar.
func (lv *LogView) renderScrollbar() string {
	if lv.filteredCount == 0 {
		return ""
	}

	height := lv.viewport.Height
	if height <= 0 {
		return ""
	}

	// Calculate thumb size and position
	thumbSize := height
	if lv.filteredCount > height {
		thumbSize = max(1, height*height/lv.filteredCount)
	}

	thumbPos := 0
	if lv.filteredCount > height {
		thumbPos = (lv.selectedIndex * (height - thumbSize)) / (lv.filteredCount - 1)
	}

	var scrollbar strings.Builder
	trackStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	thumbStyle := lipgloss.NewStyle().Foreground(ColorPrimary)

	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			scrollbar.WriteString(thumbStyle.Render(ScrollbarThumb))
		} else {
			scrollbar.WriteString(trackStyle.Render(ScrollbarTrack))
		}
		if i < height-1 {
			scrollbar.WriteString("\n")
		}
	}

	return " " + scrollbar.String()
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
