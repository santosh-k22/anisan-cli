// Package style provides a functional API for composing and applying lipgloss-based TUI styles.
package style

import "github.com/charmbracelet/lipgloss"

// Palette defines the application's color scheme.
var (
	// Base colors
	Base    = lipgloss.Color("#1e1e2e")
	Text    = lipgloss.Color("#cdd6f4")
	Subtext = lipgloss.Color("#a6adc8")
	Overlay = lipgloss.Color("#6c7086")
	Surface = lipgloss.Color("#313244")

	// Accents
	Rosewater = lipgloss.Color("#f5e0dc")
	Flamingo  = lipgloss.Color("#f2cdcd")
	Pink      = lipgloss.Color("#f5c2e7")
	Mauve     = lipgloss.Color("#cba6f7")
	Red       = lipgloss.Color("#f38ba8")
	Maroon    = lipgloss.Color("#eba0ac")
	Peach     = lipgloss.Color("#fab387")
	Yellow    = lipgloss.Color("#f9e2af")
	Green     = lipgloss.Color("#a6e3a1")
	Teal      = lipgloss.Color("#94e2d5")
	Sky       = lipgloss.Color("#89dceb")
	Sapphire  = lipgloss.Color("#74c7ec")
	Blue      = lipgloss.Color("#89b4fa")
	Lavender  = lipgloss.Color("#b4befe")

	// Semantic mappings
	AccentColor    = Mauve
	SecondaryColor = Lavender
	SuccessColor   = Green
	WarningColor   = Yellow
	ErrorColor     = Red
	HiRed          = Red
	FaintColor     = Overlay

	// UI Elements
	BorderColor       = Surface
	ActiveBorderColor = AccentColor
)
