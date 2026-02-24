package constant

import _ "embed"

// AsciiArtLogo is the application's ASCII art banner, loaded at compile time.
//
//go:embed ascii.txt
var AsciiArtLogo string
