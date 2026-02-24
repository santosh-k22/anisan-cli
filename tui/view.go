// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"strings"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/style"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

var (
	listExtraPaddingStyle = lipgloss.NewStyle().Padding(1, 2, 1, 0)
	paddingStyle          = lipgloss.NewStyle().Padding(1, 2)
)

func (b *statefulBubble) View() string {
	var output string

	switch b.state {
	case loadingState:
		output = b.viewLoading()
	case historyState:
		output = b.viewHistory()
	case sourcesState:
		output = b.viewSources()
	case searchState:
		output = b.viewSearch()
	case animesState:
		output = b.viewAnimes()
	case episodesState:
		output = b.viewEpisodes()
	case anilistSelectState:
		output = b.viewAniList()
	case malSelectState:
		output = b.viewMALList()
	case readState:
		output = b.viewRead()
	case postWatchState:
		output = b.viewPostWatch()
	case manualIDState:
		output = b.viewManualID()
	case errorState:
		output = b.viewError()
	default:
		output = "Unknown state"
	}

	return b.notifier.View(output)
}

func (b *statefulBubble) viewLoading() string {
	return b.renderLines(
		true,
		[]string{
			style.Title("Loading"),
			"",
			b.spinnerC.View() + " " + b.progressStatus,
		},
	)
}

func (b *statefulBubble) viewHistory() string {
	return listExtraPaddingStyle.Render(b.historyC.View())
}

func (b *statefulBubble) viewSources() string {
	return listExtraPaddingStyle.Render(b.sourcesC.View())
}

func (b *statefulBubble) viewSearch() string {
	lines := []string{
		style.Title("Search Anime"),
		"",
		b.inputC.View(),
	}

	return b.renderLines(true, lines)
}

func (b *statefulBubble) viewAnimes() string {
	return listExtraPaddingStyle.Render(b.animesC.View())
}

func (b *statefulBubble) viewEpisodes() string {
	return listExtraPaddingStyle.Render(b.episodesC.View())
}

func (b *statefulBubble) viewAniList() string {
	return listExtraPaddingStyle.Render(b.anilistC.View())
}

func (b *statefulBubble) viewMALList() string {
	return listExtraPaddingStyle.Render(b.malListC.View())
}

func (b *statefulBubble) viewPostWatch() string {
	return listExtraPaddingStyle.Render(b.postWatchC.View())
}

func (b *statefulBubble) viewRead() string {
	var episodeName string

	episode := b.currentPlayingEpisode
	if episode != nil {
		episodeName = episode.Name
	}

	return b.renderLines(
		true,
		[]string{
			style.Title("Now Playing"),
			"",
			style.Truncate(b.width)(fmt.Sprintf(icon.Get(icon.Progress)+" %s", style.Fg(color.Purple)(episodeName))),
			"",
			style.Truncate(b.width)(b.spinnerC.View() + " " + b.progressStatus),
		},
	)
}

func (b *statefulBubble) viewError() string {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	errorBody := errorStyle.Render(fmt.Sprintf("Critical Failure: %v", b.lastError.Error()))
	errorMsg := wrap.String(errorBody, b.width)
	return b.renderLines(
		true,
		append([]string{
			style.ErrorTitle("Error"),
			"",
			icon.Get(icon.Fail) + " An error occurred:",
			"",
		},
			errorMsg,
		),
	)
}

func (b *statefulBubble) renderLines(addHelp bool, lines []string) string {
	h := len(lines)
	l := strings.Join(lines, "\n")
	if addHelp {
		if b.height > h {
			l += strings.Repeat("\n", b.height-h)
		}
		l += b.helpC.View(b.keymap)
	}

	return paddingStyle.Render(l)
}
func (b *statefulBubble) viewManualID() string {
	lines := []string{
		style.Title("Manual ID Override"),
		"",
		"Enter MyAnimeList or AniList ID for:",
		style.Fg(color.Purple)(b.selectedAnime.Name),
		"",
		b.idInputC.View(),
		"",
		style.Faint("(Enter to confirm, Esc to cancel)"),
	}

	return b.renderLines(false, lines)
}
