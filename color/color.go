// Package color provides a curated palette of colors.
package color

import "github.com/charmbracelet/lipgloss"

// New initializes a lipgloss.Color from a string value.
func New(value string) lipgloss.Color {
	return lipgloss.Color(value)
}

// Standard ANSI 8-color palette.
var (
	Red    = New("1")
	Green  = New("2")
	Yellow = New("3")
	Blue   = New("4")
	Purple = New("5")
	Cyan   = New("6")
	White  = New("7")
	Black  = New("8")
)

// High-intensity ANSI 16-color palette extension.
var (
	HiRed    = New("9")
	HiGreen  = New("10")
	HiYellow = New("11")
	HiBlue   = New("12")
	HiPurple = New("13")
	HiCyan   = New("14")
	HiWhite  = New("15")
	HiBlack  = New("16")
)

// Hex-defined accent and semantic colors.
var (
	Orange = New("#ffb703")
	Gray   = New("#808080")
)
