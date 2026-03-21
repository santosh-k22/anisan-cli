// Package style provides a functional API for composing and applying lipgloss-based TUI styles.
package style

import (
	"github.com/charmbracelet/lipgloss"
)

// New returns an empty lipgloss.Style used as a foundation for visual composition.
func New() lipgloss.Style {
	return lipgloss.NewStyle()
}

// Pre-compiled styles to ensure zero-allocation rendering in high-frequency TUI loops.
var (
	BoldStyle      = lipgloss.NewStyle().Bold(true)
	ItalicStyle    = lipgloss.NewStyle().Italic(true)
	FaintStyle     = lipgloss.NewStyle().Faint(true)
	UnderlineStyle = lipgloss.NewStyle().Underline(true)

	// UI Layout Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(Base).
			Background(AccentColor).
			Padding(0, 1)

	ErrorTitleStyle = lipgloss.NewStyle().
			Foreground(Base).
			Background(ErrorColor).
			Padding(0, 1)

	// Invisible Cursor: A minimalist approach to selection.
	// Uses a vertical bar and bold accent instead of a heavy background block.
	SelectedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(AccentColor).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderLeftForeground(AccentColor).
				Padding(0, 0, 0, 1)

	SelectedDescStyle = lipgloss.NewStyle().
				Faint(true).
				Foreground(AccentColor).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderLeftForeground(AccentColor).
				Padding(0, 0, 0, 1)

	// Focus Dimming: Reduces cognitive load by fainting inactive components.
	DimmedStyle = lipgloss.NewStyle().Faint(true)

	// Episode Mark: Used for indicating selected episodes.
	EpisodeMarkStyle = lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
)

// Functional helpers for backward compatibility, now using pre-compiled styles.
var (
	Faint     = func(s string) string { return FaintStyle.Render(s) }
	Bold      = func(s string) string { return BoldStyle.Render(s) }
	Italic    = func(s string) string { return ItalicStyle.Render(s) }
	Underline = func(s string) string { return UnderlineStyle.Render(s) }
	Title     = func(s string) string { return TitleStyle.Render(s) }
	ErrorTitle = func(s string) string { return ErrorTitleStyle.Render(s) }
)

// Fg returns a stateless rendering function that applies the specified foreground color to a string.
func Fg(c lipgloss.TerminalColor) func(string) string {
	return func(s string) string { return lipgloss.NewStyle().Foreground(c).Render(s) }
}

// Bg returns a stateless rendering function that applies the specified background color to a string.
func Bg(c lipgloss.TerminalColor) func(string) string {
	return func(s string) string { return lipgloss.NewStyle().Background(c).Render(s) }
}

// Truncate returns a rendering function that constrains the output string to a specified maximum width.
func Truncate(max int) func(string) string {
	return func(s string) string { return lipgloss.NewStyle().Width(max).Render(s) }
}

// Tag returns a rendering function that encapsulates a string in a colored, padded tag block.
func Tag(fg, bg lipgloss.TerminalColor) func(string) string {
	return func(s string) string { return lipgloss.NewStyle().Foreground(fg).Background(bg).Padding(0, 1).Render(s) }
}
