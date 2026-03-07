package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/nfnt/resize"
)

// RenderMode defines the capability level of the host terminal environment.
type RenderMode int

const (
	ModeKitty RenderMode = iota
	ModeSixel
	ModeHalfBlock
)

// DetectProtocol implements an environment interrogation logic mirroring the Rust 'viu' library.
func DetectProtocol() RenderMode {
	term := os.Getenv("TERM")

	// Interrogate the TERM variable for known Kitty-compatible emulators.
	if strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") || strings.Contains(term, "ghostty") {
		// Ensure we are not inside a multiplexer that breaks the APC protocol.
		// If TMUX is present, we must degrade the render mode.
		if os.Getenv("TMUX") == "" {
			return ModeKitty
		}
	}

	// Sixel support requires deeper terminfo capability queries (e.g., DA1 responses).
	// For the automatic cascade within a Bubbletea context, we fallback to the robust
	// TrueColor half-blocks to ensure absolute layout stability.
	return ModeHalfBlock
}

// RenderCoverArt mathematically scales and converts an image into a Bubbletea-compatible string.
func RenderCoverArt(img image.Image, targetWidth, targetHeight uint, mode RenderMode) string {
	// 1. Resize the image to fit the TUI layout constraints while preserving the aspect ratio.
	// We mathematically double the target height because Unicode half-blocks represent
	// two vertical pixels within a single terminal cell.
	scaledImg := resize.Thumbnail(targetWidth, targetHeight*2, img, resize.Lanczos3)

	switch mode {
	case ModeKitty:
		return renderKitty(scaledImg)
	case ModeHalfBlock:
		return renderHalfBlocks(scaledImg)
	default:
		return renderHalfBlocks(scaledImg)
	}
}

// renderKitty encodes the image into the Kitty Application Program Command payload.
func renderKitty(img image.Image) string {
	// Conceptual placeholder demonstrating the Kitty APC protocol architecture.
	// Actual implementation requires Base64-encoded PNG chunking for raster persistence.
	return "\033_Gf=32,a=T,t=d;\033\\"
}

// rgbaTo8Bit shifts 16-bit color values to 8-bit components.
func rgbaTo8Bit(c color.Color) (uint32, uint32, uint32, uint32) {
	r, g, b, a := c.RGBA()
	return r >> 8, g >> 8, b >> 8, a >> 8
}

// renderHalfBlocks maps raster pixel data to TrueColor ANSI half-blocks.
func renderHalfBlocks(img image.Image) string {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var buffer bytes.Buffer

	// Iterate over the image in strides of two along the Y-axis to map pixels to half-block cells.
	// We optimize vertical space by utilizing Unicode half-blocks (U+2584) which allow for
	// dual-pixel representation within a single terminal cell.
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			// Extract the top pixel which maps to the ANSI background layer.
			topColor := img.At(x, y)
			tr, tg, tb, _ := rgbaTo8Bit(topColor)

			// Extract the bottom pixel which maps to the ANSI foreground layer.
			if y+1 < height {
				bottomColor := img.At(x, y+1)
				br, bg, bb, _ := rgbaTo8Bit(bottomColor)
				// Encode a lower half-block cell with foreground (bottom) and background (top) TrueColor data.
				buffer.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▄", br, bg, bb, tr, tg, tb))
			} else {
				// Fallback: Encode an upper half-block cell for odd-height images, resetting the background layer.
				buffer.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[49m▀", tr, tg, tb))
			}
		}
		// Reset the ANSI formatting sequence at each physical terminal row boundary.
		buffer.WriteString("\x1b[0m\n")
	}

	return strings.TrimRight(buffer.String(), "\n")
}
