// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"strings"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/history"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/style"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// listItem implements the list.Item interface, wrapping various domain models for terminal display.
type listItem struct {
	internal interface{}
	marked   bool
}

func (t *listItem) toggleMark() {
	t.marked = !t.marked
}

func (t *listItem) getMark() string {
	switch t.internal.(type) {
	case *source.Episode:
		return lipgloss.NewStyle().Bold(true).Foreground(style.AccentColor).Render(icon.Get(icon.Mark))
	case *anilist.Anime, *mal.Anime: // Unified tracker link icon rendering
		return icon.Get(icon.Link)
	case *provider.Provider:
		return icon.Get(icon.Search)
	default:
		return ""
	}
}

// Title retrieves the primary display text for the list item.
func (t *listItem) Title() (title string) {
	switch e := t.internal.(type) {
	case *source.Episode:
		var sb = strings.Builder{}

		sb.WriteString(t.FilterValue())
		if e.Volume != "" {
			sb.WriteString(" ")
			sb.WriteString(style.Faint(e.Volume))
		}

		title = sb.String()
	case *source.Anime:
		if e.Metadata.Title != "" {
			title = e.Metadata.Title
		} else {
			title = e.Name
		}
	case *mal.Anime:
		title = e.Title
	case string:
		title = e
	default:
		title = t.FilterValue()
	}

	if title != "" && t.marked {
		title = fmt.Sprintf("%s %s", title, t.getMark())
	}

	return
}

// Description retrieves the multi-line secondary metadata for the list item.
func (t *listItem) Description() (description string) {
	switch e := t.internal.(type) {
	case *source.Episode:
		description = ""
	case *source.Anime:
		var parts []string

		// Status Indicator (e.g., FINISHED, RELEASING)
		if e.Metadata.Status != "" {
			var c lipgloss.Color
			if e.Metadata.Status == "RELEASING" {
				c = style.Green
			} else {
				c = style.Subtext
			}
			// Title-case formatting for visual consistency in the list view.
			statusStr := strings.ToLower(e.Metadata.Status)
			if len(statusStr) > 0 {
				statusStr = strings.ToUpper(statusStr[:1]) + statusStr[1:]
			}
			parts = append(parts, lipgloss.NewStyle().Foreground(c).Render(statusStr))
		}

		// Dynamic Rating: Render the media score according to the active backend's numeric scale.
		// MyAnimeList utilizes a 1.0-10.0 decimal scale, whereas AniList employs a 0-100 percentage range.
		if e.Metadata.Score > 0 {
			if viper.GetString(key.TrackerBackend) == "mal" {
				parts = append(parts, lipgloss.NewStyle().Foreground(style.AccentColor).Render(fmt.Sprintf("★ %.1f", float64(e.Metadata.Score)/10.0)))
			} else {
				parts = append(parts, lipgloss.NewStyle().Foreground(style.AccentColor).Render(fmt.Sprintf("★ %d%%", e.Metadata.Score)))
			}
		}

		// Release Year
		if e.Metadata.StartDate.Year > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(style.FaintColor).Render(fmt.Sprintf("%d", e.Metadata.StartDate.Year)))
		}

		// Total Episode Count
		if e.Metadata.Episodes > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(style.FaintColor).Render(fmt.Sprintf("%d eps", e.Metadata.Episodes)))
		}

		description = strings.Join(parts, " • ")

	case *history.SavedEpisode:
		var parts []string

		// Format chronological episode position and total count.
		parts = append(parts, fmt.Sprintf("Ep: %d/%d", e.Index, e.AnimeEpisodesTotal))

		// Render media lifecycle status with semantic color coding (Green for Releasing).
		if e.Status != "" {
			var c lipgloss.Color
			if e.Status == "RELEASING" {
				c = style.Green
			} else {
				c = style.Subtext
			}
			statusStr := strings.ToLower(e.Status)
			if len(statusStr) > 0 {
				statusStr = strings.ToUpper(statusStr[:1]) + statusStr[1:]
			}
			parts = append(parts, lipgloss.NewStyle().Foreground(c).Render(statusStr))
		}

		// Scale and display user/aggregate score based on the active tracking backend (MAL uses 10.0, Anilist uses %).
		if e.Score > 0 {
			if viper.GetString(key.TrackerBackend) == "mal" {
				parts = append(parts, lipgloss.NewStyle().Foreground(style.AccentColor).Render(fmt.Sprintf("★ %.1f", float64(e.Score)/10.0)))
			} else {
				parts = append(parts, lipgloss.NewStyle().Foreground(style.AccentColor).Render(fmt.Sprintf("★ %d%%", e.Score)))
			}
		}

		// Genres
		if len(e.Genres) > 0 {
			// Limit genres to prevent overly long lines
			displayGenres := e.Genres
			if len(displayGenres) > 3 {
				displayGenres = displayGenres[:3]
			}
			parts = append(parts, lipgloss.NewStyle().Foreground(style.FaintColor).Render(strings.Join(displayGenres, ", ")))
		}

		// Progress
		completionThreshold := viper.GetFloat64(key.PlayerCompletionPercentage)
		if completionThreshold <= 0 {
			completionThreshold = 80.0
		}
		if e.WatchedPercentage > 0 && e.WatchedPercentage < completionThreshold {
			parts = append(parts, lipgloss.NewStyle().Foreground(style.Yellow).Render(fmt.Sprintf("%.0f%%", e.WatchedPercentage)))
		} else if e.WatchedPercentage >= completionThreshold {
			parts = append(parts, lipgloss.NewStyle().Foreground(style.Green).Render("Watched"))
		}

		description = strings.Join(parts, " • ")
	case *provider.Provider:
		sb := strings.Builder{}
		if e.IsCustom {
			sb.WriteString("Lua Extension")
		} else {
			sb.WriteString("Built-in Provider")
		}

		if e.UsesHeadless {
			sb.WriteString(" (Requires Headless Chrome)")
		}

		description = sb.String()
	case *anilist.Anime:
		description = e.SiteURL
	case *mal.Anime:
		description = fmt.Sprintf("https://myanimelist.net/anime/%d", e.ID)
	case string:
		description = ""
	}

	return
}

// FilterValue returns the string used for real-time list filtering and searching.
func (t *listItem) FilterValue() string {
	switch e := t.internal.(type) {
	case *source.Episode:
		return e.Name
	case *source.Anime:
		// Filter by both name and english title if available
		if e.Metadata.Title != "" && e.Metadata.Title != e.Name {
			return e.Name + " " + e.Metadata.Title
		}
		return e.Name
	case *history.SavedEpisode:
		parts := []string{e.AnimeName}
		if e.Status != "" {
			parts = append(parts, e.Status)
		}
		parts = append(parts, e.Genres...)
		return strings.Join(parts, " ")
	case *anilist.Anime:
		return e.Name()
	case *provider.Provider:
		return e.Name
	case *mal.Anime:
		return e.Title
	case string:
		return e
	default:
		return ""
	}
}
