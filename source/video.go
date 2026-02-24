// Package source defines the domain models and interfaces for media discovery and retrieval.
package source

// Video represents a streamable video resource.
type Video struct {
	// Direct URL to the stream/file.
	URL string `json:"url"`
	// Quality label (e.g. "1080p", "720p").
	Quality string `json:"quality"`
	// File extension (e.g. "mp4", "m3u8").
	Extension string `json:"extension"`
	// HTTP headers required to stream.
	Headers map[string]string `json:"headers"`
	// Ordering index.
	Index uint16 `json:"index"`
}

// String returns the quality or URL for display.
func (v *Video) String() string {
	if v.Quality != "" {
		return v.Quality
	}
	return v.URL
}
