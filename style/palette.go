// Package style provides a functional API for composing and applying lipgloss-based TUI styles.
package style

import "github.com/charmbracelet/lipgloss"

// Palette defines the application's color scheme.
var (
	// Adaptive Colors (Dark: Mocha, Light: Latte)
	Base      = lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"}
	Text      = lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	Subtext   = lipgloss.AdaptiveColor{Light: "#5c5f77", Dark: "#a6adc8"}
	Overlay   = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#6c7086"}
	Surface   = lipgloss.AdaptiveColor{Light: "#ccd0da", Dark: "#313244"}
	Rosewater = lipgloss.AdaptiveColor{Light: "#dc8a78", Dark: "#f5e0dc"}
	Flamingo  = lipgloss.AdaptiveColor{Light: "#dd7878", Dark: "#f2cdcd"}
	Pink      = lipgloss.AdaptiveColor{Light: "#ea76cb", Dark: "#f5c2e7"}
	Mauve     = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	Red       = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	Maroon    = lipgloss.AdaptiveColor{Light: "#e64553", Dark: "#eba0ac"}
	Peach     = lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}
	Yellow    = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	Green     = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	Teal      = lipgloss.AdaptiveColor{Light: "#179287", Dark: "#94e2d5"}
	Sky       = lipgloss.AdaptiveColor{Light: "#04a5e5", Dark: "#89dceb"}
	Sapphire  = lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}
	Blue      = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	Lavender  = lipgloss.AdaptiveColor{Light: "#7287fd", Dark: "#b4befe"}

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
