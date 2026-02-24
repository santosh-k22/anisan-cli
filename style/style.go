// Package style provides a functional API for composing and applying lipgloss-based TUI styles.
package style

import (
	"github.com/anisan-cli/anisan/color"
	"github.com/charmbracelet/lipgloss"
)

// New returns an empty lipgloss.Style used as a foundation for visual composition.
func New() lipgloss.Style {
	return lipgloss.NewStyle()
}

// Colored initializes a new style with the specified foreground and background colors.
func Colored(fg, bg lipgloss.Color) lipgloss.Style {
	return New().Foreground(fg).Background(bg)
}

// NewColored is an alias for Colored (backward compat).
var NewColored = Colored

// Fg returns a stateless rendering function that applies the specified foreground color to a string.
func Fg(c lipgloss.Color) func(string) string {
	return func(s string) string { return Colored(c, "").Render(s) }
}

// Bg returns a stateless rendering function that applies the specified background color to a string.
func Bg(c lipgloss.Color) func(string) string {
	return func(s string) string { return Colored("", c).Render(s) }
}

// Truncate returns a rendering function that constrains the output string to a specified maximum width.
func Truncate(max int) func(string) string {
	return func(s string) string { return New().Width(max).Render(s) }
}

// Standard Text Transformation Helpers - these functions apply common typographic styles like bold or italics.
var (
	Faint     = func(s string) string { return New().Faint(true).Render(s) }
	Bold      = func(s string) string { return New().Bold(true).Render(s) }
	Italic    = func(s string) string { return New().Italic(true).Render(s) }
	Underline = func(s string) string { return New().Underline(true).Render(s) }
)

// Palette defines the canonical color scheme used across the Terminal User Interface.
var Title = func(s string) string {
	return Colored(color.New("230"), color.New("62")).Padding(0, 1).Render(s)
}

// ErrorTitle renders a visually highlighted banner using dominant error status colors.
var ErrorTitle = func(s string) string {
	return Colored(color.New("230"), color.Red).Padding(0, 1).Render(s)
}

// Tag returns a rendering function that encapsulates a string in a colored, padded tag block.
func Tag(fg, bg lipgloss.Color) func(string) string {
	return func(s string) string { return Colored(fg, bg).Padding(0, 1).Render(s) }
}
