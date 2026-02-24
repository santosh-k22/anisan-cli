// Package mini implements a lightweight, minimalist interface for anime search and playback.
package mini

import (
	"os"

	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/util"
	"github.com/samber/lo"
)

var (
	truncateAt = 100
)

type Options struct {
	Continue bool
}

type mini struct {
	width, height int

	state         state
	statesHistory util.Stack[state]

	selectedSource source.Source

	cachedAnimes   map[string][]*source.Anime
	cachedEpisodes map[string][]*source.Episode

	query            string
	selectedAnime    *source.Anime
	selectedEpisodes []*source.Episode
}

func newMini() *mini {
	return &mini{
		statesHistory:  util.Stack[state]{},
		cachedAnimes:   make(map[string][]*source.Anime),
		cachedEpisodes: make(map[string][]*source.Episode),
	}
}

func (m *mini) previousState() {
	if m.statesHistory.Len() > 0 {
		m.setState(m.statesHistory.Pop())
	}
}

func (m *mini) setState(s state) {
	m.state = s
}

func (m *mini) newState(s state) {
	if m.state == s {
		return
	}

	if !lo.Contains([]state{}, m.state) {
		m.statesHistory.Push(m.state)
	}

	m.setState(s)
}

func Run(options *Options) error {
	m := newMini()
	m.state = sourceSelectState
	if options.Continue {
		m.state = historySelectState
	}

	if w, h, err := util.TerminalSize(); err == nil {
		m.width, m.height = w, h
		truncateAt = w
	}

	var err error

	for {
		if m.handleState() != nil {
			return err
		}
	}
}

func (m *mini) handleState() error {
	switch m.state {
	case historySelectState:
		return m.handleHistorySelectState()
	case sourceSelectState:
		return m.handleSourceSelectState()
	case animesSearchState:
		return m.handleAnimeSearchState()
	case animeSelectState:
		return m.handleAnimeSelectState()
	case episodeSelectState:
		return m.handleEpisodeSelectState()
	case episodeReadState:
		return m.handleEpisodeReadState()
	case quitState:
		os.Exit(0)
	}

	return nil
}
