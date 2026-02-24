// Package icon provides a flexible multi-variant rendering engine for UI symbols and feedback indicators.
//
// Icons can be displayed as emoji, nerd-font glyphs, plain ASCII, kaomoji,
// or Unicode squares depending on user preference.
package icon

import (
	"github.com/anisan-cli/anisan/key"
	"github.com/spf13/viper"
)

// Visual Variant Constants - these define the supported aesthetic styles for icon rendering.
const (
	emoji   = "emoji"
	nerd    = "nerd"
	plain   = "plain"
	kaomoji = "kaomoji"
	squares = "squares"
)

// AvailableVariants returns a slice of all registered icon style identifiers.
func AvailableVariants() []string {
	return []string{emoji, nerd, plain, kaomoji, squares}
}

// iconDef encapsulates the visual representations of a single UI symbol across all supported variants.
type iconDef struct {
	emoji   string
	nerd    string
	plain   string
	kaomoji string
	squares string
}

// Get retrieves the visual representation for the receiver Def based on the global icons variant configuration.
func (d *iconDef) Get() string {
	switch viper.GetString(key.IconsVariant) {
	case emoji:
		return d.emoji
	case nerd:
		return d.nerd
	case plain:
		return d.plain
	case kaomoji:
		return d.kaomoji
	case squares:
		return d.squares
	default:
		return ""
	}
}

// Get returns the rendered string for a specified Icon identifier from the global registry.
func Get(i Icon) string {
	return icons[i].Get()
}
