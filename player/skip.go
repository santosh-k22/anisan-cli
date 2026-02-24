// Package player defines a unified abstraction layer for media playback engines.
package player

import (
	"fmt"

	"github.com/anisan-cli/anisan/aniskip"
	"github.com/anisan-cli/anisan/log"
)

// Skipper handles auto-skipping of intros and outros.
type Skipper struct {
	Times *aniskip.SkipTimes
	mpv   *MPV
}

// NewSkipper creates a new Skipper instance.
func NewSkipper(mpv *MPV, times *aniskip.SkipTimes) *Skipper {
	return &Skipper{
		Times: times,
		mpv:   mpv,
	}
}

// Check inspects the current playback position and skips if inside a skip interval.
// Returns true if a skip was performed.
func (s *Skipper) Check(pos float64) (bool, error) {
	if s.Times == nil {
		return false, nil
	}

	// Check Intro
	if s.Times.HasIntro && pos >= s.Times.Opening.Start && pos < s.Times.Opening.End {
		log.Infof("Skipping intro: %v -> %v", pos, s.Times.Opening.End)
		if err := s.mpv.Seek(s.Times.Opening.End); err != nil {
			return false, fmt.Errorf("skip intro seek: %w", err)
		}
		return true, nil
	}

	// Check Outro
	if s.Times.HasOutro && pos >= s.Times.Ending.Start && pos < s.Times.Ending.End {
		log.Infof("Skipping outro: %v -> %v", pos, s.Times.Ending.End)
		// For outro, we usually skip to the end of the interval, which might be the end of the episode
		// or the start of a preview.
		if err := s.mpv.Seek(s.Times.Ending.End); err != nil {
			return false, fmt.Errorf("skip outro seek: %w", err)
		}
		return true, nil
	}

	return false, nil
}

// ApplyChapters sends chapter markers to the player for visual feedback.
func (s *Skipper) ApplyChapters() error {
	if s.Times == nil {
		return nil
	}

	var chapters []map[string]interface{}

	// Always start with Pre-Opening/Main at 0
	chapters = append(chapters, map[string]interface{}{
		"title": "Part A",
		"time":  0.0,
	})

	if s.Times.HasIntro {
		// Update previous chapter name if it was just "Part A" and intro is at start?
		// Actually, let's just add Opening. MPV sorts by time? No, order matters.
		// Chapter marker logic:
		// 1. Pre-Opening (0.0)
		// 2. Opening (Start)
		// 3. Main (End)

		chapters = append(chapters, map[string]interface{}{
			"title": "Opening",
			"time":  s.Times.Opening.Start,
		})
		chapters = append(chapters, map[string]interface{}{
			"title": "Part B",
			"time":  s.Times.Opening.End,
		})
	}

	if s.Times.HasOutro {
		chapters = append(chapters, map[string]interface{}{
			"title": "Ending",
			"time":  s.Times.Ending.Start,
		})
		chapters = append(chapters, map[string]interface{}{
			"title": "Preview / Next",
			"time":  s.Times.Ending.End,
		})
	}

	return s.mpv.SetChapters(chapters)
}
