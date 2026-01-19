// Package tui provides the terminal user interface components.
package tui

import (
	"fmt"
	"strings"

	"github.com/Expert21/argus/internal/aggregate"
	"github.com/charmbracelet/lipgloss"
)

// Sidebar displays the list of log sources and their health status.
type Sidebar struct {
	// aggregator provides source information
	aggregator *aggregate.Aggregator

	// width is the fixed width of the sidebar
	width int

	// height is the available height
	height int

	// focused indicates if this component has focus
	focused bool

	// selectedIndex is the currently highlighted source
	selectedIndex int

	// activeFilter is the source currently being filtered (empty = show all)
	activeFilter string

	// sourceNames caches the source list for stable ordering
	sourceNames []string
}

// NewSidebar creates a new sidebar component.
func NewSidebar(agg *aggregate.Aggregator) *Sidebar {
	return &Sidebar{
		aggregator:   agg,
		width:        25,
		activeFilter: "", // Empty means "All Sources"
	}
}

// SetSize updates the sidebar dimensions.
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetFocused sets the focus state.
func (s *Sidebar) SetFocused(focused bool) {
	s.focused = focused
}

// RefreshSources updates the cached source list.
func (s *Sidebar) RefreshSources() {
	s.sourceNames = s.aggregator.GetSources()
}

// View renders the sidebar.
func (s *Sidebar) View() string {
	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("📋 Sources")
	content.WriteString(title)
	content.WriteString("\n\n")

	// "All Sources" option
	allLabel := "All Sources"
	if s.activeFilter == "" {
		// Currently filtering by all
		if s.selectedIndex == 0 && s.focused {
			content.WriteString(SourceItemSelectedStyle.Render("▸ ✓ " + allLabel))
		} else if s.selectedIndex == 0 {
			content.WriteString(SourceItemStyle.Render("▹ ✓ " + allLabel))
		} else {
			content.WriteString(lipgloss.NewStyle().Foreground(ColorSuccess).Render("  ✓ " + allLabel))
		}
	} else {
		if s.selectedIndex == 0 && s.focused {
			content.WriteString(SourceItemSelectedStyle.Render("▸   " + allLabel))
		} else if s.selectedIndex == 0 {
			content.WriteString(SourceItemStyle.Render("▹   " + allLabel))
		} else {
			content.WriteString(SourceItemStyle.Render("    " + allLabel))
		}
	}
	content.WriteString("\n")

	// Separator
	content.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render("  ─────────────────"))
	content.WriteString("\n")

	// Get sources and their health
	sources := s.aggregator.GetSources()
	health := s.aggregator.GetSourceHealth()

	if len(sources) == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true).
			Render("  No sources"))
	} else {
		for i, name := range sources {
			// Adjust index (0 is "All Sources")
			itemIndex := i + 1

			// Health indicator
			var indicator string
			if healthy, ok := health[name]; ok && healthy {
				indicator = SourceHealthyStyle.Render("●")
			} else {
				indicator = SourceUnhealthyStyle.Render("●")
			}

			// Check if this is the active filter
			isActive := s.activeFilter == name
			var checkMark string
			if isActive {
				checkMark = "✓"
			} else {
				checkMark = " "
			}

			// Build the line
			var line string
			displayName := truncateStr(name, s.width-10)

			if itemIndex == s.selectedIndex && s.focused {
				line = SourceItemSelectedStyle.Render(fmt.Sprintf("▸ %s %s %s", checkMark, indicator, displayName))
			} else if itemIndex == s.selectedIndex {
				line = SourceItemStyle.Render(fmt.Sprintf("▹ %s %s %s", checkMark, indicator, displayName))
			} else if isActive {
				line = lipgloss.NewStyle().Foreground(ColorSuccess).Render(fmt.Sprintf("  %s %s %s", checkMark, indicator, displayName))
			} else {
				line = SourceItemStyle.Render(fmt.Sprintf("  %s %s %s", checkMark, indicator, displayName))
			}

			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	// Spacer to push stats to bottom
	usedLines := 4 + len(sources) // title + all + separator + sources
	remainingLines := s.height - usedLines - 6
	if remainingLines > 0 {
		content.WriteString(strings.Repeat("\n", remainingLines))
	}

	// Stats at bottom
	content.WriteString("\n")
	countStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true)
	content.WriteString(countStyle.Render(
		fmt.Sprintf("Events: %d", s.aggregator.EntryCount()),
	))

	// Apply the sidebar border style
	style := SidebarStyle.Width(s.width).Height(s.height)
	if s.focused {
		style = SidebarFocusedStyle.Width(s.width).Height(s.height)
	}

	return style.Render(content.String())
}

// MoveUp moves selection up.
func (s *Sidebar) MoveUp() {
	if s.selectedIndex > 0 {
		s.selectedIndex--
	}
}

// MoveDown moves selection down.
func (s *Sidebar) MoveDown() {
	sources := s.aggregator.GetSources()
	maxIndex := len(sources) // +1 for "All Sources" but already 0-indexed so just len
	if s.selectedIndex < maxIndex {
		s.selectedIndex++
	}
}

// Select activates the currently highlighted source as the filter.
// Returns the new active filter (empty string means "All Sources").
func (s *Sidebar) Select() string {
	sources := s.aggregator.GetSources()

	if s.selectedIndex == 0 {
		// "All Sources" selected
		s.activeFilter = ""
	} else {
		// Specific source selected
		idx := s.selectedIndex - 1
		if idx >= 0 && idx < len(sources) {
			s.activeFilter = sources[idx]
		}
	}

	return s.activeFilter
}

// ActiveFilter returns the currently active source filter.
func (s *Sidebar) ActiveFilter() string {
	return s.activeFilter
}

// SelectedSource returns the currently highlighted source name.
func (s *Sidebar) SelectedSource() string {
	sources := s.aggregator.GetSources()
	if s.selectedIndex == 0 {
		return "" // All sources
	}
	idx := s.selectedIndex - 1
	if idx >= 0 && idx < len(sources) {
		return sources[idx]
	}
	return ""
}
