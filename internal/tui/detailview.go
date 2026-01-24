// Package tui provides the terminal user interface components.
package tui

import (
	"fmt"
	"strings"

	"github.com/Expert21/argus/internal/ingest"
	"github.com/charmbracelet/lipgloss"
)

// LogDetailView displays the full details of a selected log entry.
type LogDetailView struct {
	// entry is the currently displayed log entry
	entry *ingest.LogEntry

	// width and height of the view
	width, height int

	// focused indicates if this view has focus
	focused bool
}

// NewLogDetailView creates a new log detail view.
func NewLogDetailView() *LogDetailView {
	return &LogDetailView{}
}

// SetSize updates the view dimensions.
func (dv *LogDetailView) SetSize(width, height int) {
	dv.width = width
	dv.height = height
}

// SetFocused sets the focus state.
func (dv *LogDetailView) SetFocused(focused bool) {
	dv.focused = focused
}

// SetEntry updates the displayed entry.
func (dv *LogDetailView) SetEntry(entry *ingest.LogEntry) {
	dv.entry = entry
}

// View renders the log detail view.
func (dv *LogDetailView) View() string {
	// Hide if width is too small
	if dv.width <= 0 {
		return ""
	}

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("📋 Log Detail")

	var content strings.Builder

	if dv.entry == nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true).
			Render("\n  Select a log entry to view details.\n"))
	} else {
		contentWidth := dv.width - 6
		if contentWidth < 20 {
			contentWidth = 20
		}

		// Timestamp
		content.WriteString(dv.renderField("Timestamp", dv.entry.Timestamp.Format("2006-01-02 15:04:05.000")))
		content.WriteString("\n")

		// Level with color
		levelStr := LevelStyle(dv.entry.Level.String()).Render(dv.entry.Level.String())
		content.WriteString(dv.renderFieldStyled("Level", levelStr))
		content.WriteString("\n")

		// Source
		content.WriteString(dv.renderField("Source", dv.entry.Source))
		content.WriteString("\n")

		// Ingestor
		if dv.entry.IngestorName != "" && dv.entry.IngestorName != dv.entry.Source {
			content.WriteString(dv.renderField("Ingestor", dv.entry.IngestorName))
			content.WriteString("\n")
		}

		// Separator
		content.WriteString(lipgloss.NewStyle().
			Foreground(ColorBorder).
			Render(strings.Repeat("─", min(contentWidth, 30))))
		content.WriteString("\n\n")

		// Message (word-wrapped)
		content.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Render("Message:"))
		content.WriteString("\n")
		content.WriteString(dv.wrapText(dv.entry.Message, contentWidth))
		content.WriteString("\n\n")

		// Raw line (if different from message)
		if dv.entry.Raw != "" && dv.entry.Raw != dv.entry.Message {
			content.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Render("Raw:"))
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().
				Foreground(ColorDebug).
				Render(dv.wrapText(dv.entry.Raw, contentWidth)))
			content.WriteString("\n\n")
		}

		// Metadata
		if len(dv.entry.Metadata) > 0 {
			content.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Render("Metadata:"))
			content.WriteString("\n")
			for key, value := range dv.entry.Metadata {
				keyStyle := lipgloss.NewStyle().Foreground(ColorAccent)
				valStr := fmt.Sprintf("%v", value)
				if len(valStr) > contentWidth-len(key)-4 {
					valStr = valStr[:contentWidth-len(key)-7] + "..."
				}
				content.WriteString(fmt.Sprintf("  %s: %s\n",
					keyStyle.Render(key),
					valStr))
			}
		}
	}

	// Combine header and content
	inner := lipgloss.JoinVertical(lipgloss.Left,
		header,
		content.String(),
	)

	// Apply border style
	style := LogDetailStyle.Width(dv.width).Height(dv.height)
	if dv.focused {
		style = LogDetailFocusedStyle.Width(dv.width).Height(dv.height)
	}

	return style.Render(inner)
}

// renderField renders a label: value pair.
func (dv *LogDetailView) renderField(label, value string) string {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary)
	valueStyle := lipgloss.NewStyle().
		Foreground(ColorForeground)
	return fmt.Sprintf("%s %s", labelStyle.Render(label+":"), valueStyle.Render(value))
}

// renderFieldStyled renders a label with pre-styled value.
func (dv *LogDetailView) renderFieldStyled(label, styledValue string) string {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary)
	return fmt.Sprintf("%s %s", labelStyle.Render(label+":"), styledValue)
}

// wrapText wraps text to the given width.
func (dv *LogDetailView) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)
		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}
		result.WriteString(word)
		lineLen += wordLen

		// Avoid trailing space issues
		_ = i
	}

	return result.String()
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
