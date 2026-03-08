package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"strings"

	"github.com/nfnt/resize"
)

// RenderMode defines the capability level of the host terminal environment.
type RenderMode int

const (
	// ModeKitty utilizes the Kitty terminal graphics protocol for high-performance direct image rendering.
	ModeKitty RenderMode = iota
	ModeSixel
	ModeHalfBlock
)

// DetectProtocol implements an environment interrogation logic mirroring the Rust 'viu' library.
// Currently returns ModeHalfBlock uniformly to ensure robust TrueColor ASCII layout stability
// across all environments until native Kitty protocol (Base64 PNG chunking) is fully implemented.
func DetectProtocol() RenderMode {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	tmux := os.Getenv("TMUX")

	// If we're inside tmux, kitty graphics protocol generally won't work out-of-the-box
	// without bypass sequences, so default to half-blocks for safety and stability.
	if tmux != "" {
		return ModeHalfBlock
	}

	if strings.Contains(term, "kitty") || termProgram == "WezTerm" || termProgram == "ghostty" {
		return ModeKitty
	}

	return ModeHalfBlock
}

// RenderCoverArt mathematically scales and converts an image into a Bubbletea-compatible string.
func RenderCoverArt(img image.Image, targetWidth, targetHeight uint, mode RenderMode) string {
	switch mode {
	case ModeKitty:
		// Do NOT resize the image for Kitty. Let the terminal emulator scale the raw data natively.
		// We still pass the cell dimensions (targetWidth/targetHeight) so Lipgloss can reserve the layout space.
		return renderKitty(img, int(targetWidth), int(targetHeight))

	case ModeHalfBlock:
		// ONLY resize if we are using the half-block text fallback.
		// We mathematically double the target height because Unicode half-blocks represent
		// two vertical pixels within a single terminal cell.
		scaledImg := resize.Thumbnail(targetWidth, targetHeight*2, img, resize.Lanczos3)
		return renderHalfBlocks(scaledImg)

	default:
		scaledImg := resize.Thumbnail(targetWidth, targetHeight*2, img, resize.Lanczos3)
		return renderHalfBlocks(scaledImg)
	}
}

// renderKitty encodes the image into chunked Kitty Application Program Command payloads.
func renderKitty(img image.Image, widthInCells, heightInCells int) string {
	var buf bytes.Buffer
	// 1. Encode the image into PNG format
	err := png.Encode(&buf, img)
	if err != nil {
		return ""
	}

	// 2. Base64 encode the binary payload
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	var result strings.Builder

	// Invalidate and delete any previously rendered Kitty images to prevent overlapping.
	result.WriteString("\033_Ga=d,d=A\033\\")

	chunkSize := 4096

	// 3. Emit chunked APC sequences (\033_G...;\033\\)
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		m := 1 // 'm=1' tells the terminal more chunks are coming

		// If this is the final chunk
		if end >= len(b64) {
			end = len(b64)
			m = 0 // 'm=0' tells the terminal this is the final chunk
		}

		chunk := b64[i:end]

		if i == 0 {
			// First chunk must include format (f=100 for PNG), action (a=T for transmit)
			// display action (d=a for place and display), cursor action (C=1 for place at cursor)
			// columns (c=w), and rows (r=h).
			result.WriteString(fmt.Sprintf("\033_Ga=T,f=100,d=a,C=1,c=%d,r=%d,m=%d;%s\033\\",
				widthInCells, heightInCells, m, chunk))
		} else {
			// Subsequent chunks only need the 'm' flag
			result.WriteString(fmt.Sprintf("\033_Gm=%d;%s\033\\", m, chunk))
		}
	}

	return result.String()
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
