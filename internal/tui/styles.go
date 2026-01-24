// Package tui provides the terminal user interface components.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// GO SYNTAX LESSON #40: Lipgloss Styling
// ======================================
// Lipgloss is CSS-like styling for terminal UIs.
// You create styles with lipgloss.NewStyle() and chain methods.
// Styles are immutable - each method returns a NEW style.
//
// Common methods:
// - Foreground(color) / Background(color)
// - Bold(bool) / Italic(bool) / Underline(bool)
// - Padding(top, right, bottom, left) / Margin(...)
// - Border(style) / BorderForeground(color)
// - Width(n) / Height(n) / MaxWidth(n) / MaxHeight(n)
// - Align(position) - lipgloss.Left, Center, Right

// Color palette for Argus - Dark theme inspired by your Minerva rice
var (
	// Base colors
	ColorBackground  = lipgloss.Color("#0d1117")
	ColorForeground  = lipgloss.Color("#c9d1d9")
	ColorBorder      = lipgloss.Color("#30363d")
	ColorBorderFocus = lipgloss.Color("#58a6ff")

	// Accent colors
	ColorPrimary   = lipgloss.Color("#58a6ff") // Blue
	ColorSecondary = lipgloss.Color("#8b949e") // Gray
	ColorAccent    = lipgloss.Color("#a371f7") // Purple

	// Status colors
	ColorSuccess = lipgloss.Color("#3fb950") // Green
	ColorWarning = lipgloss.Color("#d29922") // Yellow/Orange
	ColorError   = lipgloss.Color("#f85149") // Red
	ColorInfo    = lipgloss.Color("#58a6ff") // Blue

	// Log level colors
	ColorDebug     = lipgloss.Color("#6e7681") // Dim gray
	ColorNotice    = lipgloss.Color("#79c0ff") // Light blue
	ColorCritical  = lipgloss.Color("#ff7b72") // Light red
	ColorAlert     = lipgloss.Color("#ffa657") // Orange
	ColorEmergency = lipgloss.Color("#f85149") // Bright red
)

// Component styles

// TitleStyle is used for the main header
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPrimary).
	Background(ColorBackground).
	Padding(0, 1).
	MarginBottom(1)

// SidebarStyle is for the source list panel
var SidebarStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorBorder).
	Padding(1, 2).
	MarginRight(1)

// SidebarFocusedStyle is sidebar when focused
var SidebarFocusedStyle = SidebarStyle.Copy().
	BorderForeground(ColorBorderFocus)

// LogViewStyle is for the main log display
var LogViewStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorBorder).
	Padding(0, 1)

// LogViewFocusedStyle is log view when focused
var LogViewFocusedStyle = LogViewStyle.Copy().
	BorderForeground(ColorBorderFocus)

// StatusBarStyle is for the bottom status bar
var StatusBarStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary).
	Background(lipgloss.Color("#161b22")).
	Padding(0, 1).
	MarginTop(1)

// StatusTextStyle is for status messages
var StatusTextStyle = lipgloss.NewStyle().
	Foreground(ColorForeground)

// StatusLiveStyle is for "LIVE" indicator
var StatusLiveStyle = lipgloss.NewStyle().
	Foreground(ColorSuccess).
	Bold(true)

// StatusPausedStyle is for "PAUSED" indicator
var StatusPausedStyle = lipgloss.NewStyle().
	Foreground(ColorWarning).
	Bold(true)

// HelpStyle is for keybinding hints
var HelpStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary).
	Italic(true)

// Log level styles

// LevelStyle returns the appropriate style for a log level string.
func LevelStyle(level string) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true).Width(7).Align(lipgloss.Center)

	switch level {
	case "DEBUG":
		return base.Foreground(ColorDebug)
	case "INFO":
		return base.Foreground(ColorInfo)
	case "NOTICE":
		return base.Foreground(ColorNotice)
	case "WARN":
		return base.Foreground(ColorWarning)
	case "ERROR":
		return base.Foreground(ColorError)
	case "CRIT":
		return base.Foreground(ColorCritical)
	case "ALERT":
		return base.Foreground(ColorAlert)
	case "EMERG":
		return base.Foreground(ColorEmergency).Background(ColorError)
	default:
		return base.Foreground(ColorSecondary)
	}
}

// SourceItemStyle is for source list items
var SourceItemStyle = lipgloss.NewStyle().
	Foreground(ColorForeground).
	PaddingLeft(2)

// SourceItemSelectedStyle is for the selected source
var SourceItemSelectedStyle = SourceItemStyle.Copy().
	Foreground(ColorPrimary).
	Bold(true).
	PaddingLeft(0)

// SourceHealthyStyle is the indicator for healthy sources
var SourceHealthyStyle = lipgloss.NewStyle().
	Foreground(ColorSuccess)

// SourceUnhealthyStyle is the indicator for unhealthy sources
var SourceUnhealthyStyle = lipgloss.NewStyle().
	Foreground(ColorError)

// TimestampStyle is for log timestamps
var TimestampStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary)

// SourceNameStyle is for log source names
var SourceNameStyle = lipgloss.NewStyle().
	Foreground(ColorAccent).
	Width(15)

// MessageStyle is for log messages
var MessageStyle = lipgloss.NewStyle().
	Foreground(ColorForeground)

// KeywordStyles for syntax highlighting in messages
var (
	KeywordErrorStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	KeywordSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	KeywordSecurityStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	KeywordIPStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#79c0ff"))
)

// LogDetailStyle is for the log detail panel
var LogDetailStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorBorder).
	Padding(1, 2)

// LogDetailFocusedStyle is log detail panel when focused
var LogDetailFocusedStyle = LogDetailStyle.Copy().
	BorderForeground(ColorBorderFocus)

// LogEntrySelectedStyle is for the currently selected log entry
var LogEntrySelectedStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1f2937")).
	Foreground(ColorForeground).
	Bold(true)

// Scrollbar characters
const (
	ScrollbarTrack = "░"
	ScrollbarThumb = "▇"
)
