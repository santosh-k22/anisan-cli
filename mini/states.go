// Package mini implements a lightweight, minimalist interface for anime search and playback.
package mini

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/anisan-cli/anisan/history"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/util"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type state int

const (
	animesSearchState state = iota + 1
	animeSelectState
	sourceSelectState
	episodeSelectState
	episodeReadState
	historySelectState
	quitState
)

func (m *mini) handleSourceSelectState() error {
	var err error

	if name := viper.GetString(key.DefaultSources); name != "" {
		p, ok := provider.Get(name)
		if !ok {
			return fmt.Errorf("unknown source \"%s\"", name)
		}

		m.selectedSource, err = p.CreateSource()
		if err != nil {
			return err
		}
	} else {
		var providers []*provider.Provider
		providers = append(providers, provider.Builtins()...)
		providers = append(providers, provider.Customs()...)

		slices.SortFunc(providers, func(a, b *provider.Provider) int {
			return strings.Compare(a.String(), b.String())
		})

		title("Select Source")
		b, p, err := menu(providers)
		if err != nil {
			return err
		}

		if quit.eq(b) {
			m.newState(quitState)
			return nil
		}

		erase := progress("Initializing Source..")
		m.selectedSource, err = p.CreateSource()
		if err != nil {
			return err
		}
		erase()
	}

	m.newState(animesSearchState)
	return nil
}

func (m *mini) handleAnimeSearchState() error {
	var searchLoop func() error
	title("Search Anime")

	searchLoop = func() error {
		in, err := getInput(func(s string) bool {
			return s != ""
		})

		if err != nil {
			return err
		}

		query := in.value

		erase := progress("Searching Query..")
		m.cachedAnimes[query], err = m.selectedSource.Search(query)
		max := lo.Min([]int{len(m.cachedAnimes[query]), viper.GetInt(key.MiniSearchLimit)})
		m.cachedAnimes[query] = m.cachedAnimes[query][:max]
		erase()

		if len(m.cachedAnimes[query]) == 0 {
			fail("No search results found")
			return searchLoop()
		}

		m.query = query
		m.newState(animeSelectState)
		return err
	}

	return searchLoop()
}

func (m *mini) handleAnimeSelectState() error {
	var err error
	title("Query Results >>")
	b, p, err := menu(m.cachedAnimes[m.query])
	if err != nil {
		return err
	}

	if quit.eq(b) {
		m.newState(quitState)
		return nil
	}

	m.selectedAnime = p
	m.newState(episodeSelectState)
	return err
}

func (m *mini) handleEpisodeSelectState() error {
	var err error

	erase := progress("Searching Episodes..")
	m.cachedEpisodes[m.selectedAnime.URL], err = m.selectedSource.EpisodesOf(m.selectedAnime)
	erase()
	if err != nil {
		return err
	}

	episodes := m.cachedEpisodes[m.selectedAnime.URL]

	if len(episodes) == 0 {
		fail("No episodes found")
		m.selectedAnime = nil
		m.newState(animeSelectState)
		return nil
	}

	title(fmt.Sprintf("To specify a range, use: start_number end_number (Episodes: 1-%d)", len(episodes)))
	oneEpisodeInput := regexp.MustCompile(`^\d+$`)
	rangeInput := regexp.MustCompile(`^\d+ \d+$`)
	in, err := getInput(func(s string) bool {
		var err error

		switch {
		case rangeInput.MatchString(s):
			var a, b int64
			{
				l := strings.Split(s, " ")
				a, err = strconv.ParseInt(l[0], 10, 16)
				if err != nil {
					return false
				}

				b, err = strconv.ParseInt(l[1], 10, 16)
				if err != nil {
					return false
				}
			}

			return a < b && 0 < a && int(a) < len(episodes) && int(b) <= len(episodes)
		case oneEpisodeInput.MatchString(s):
			var a int64
			a, err = strconv.ParseInt(s, 10, 16)
			if err != nil {
				return false
			}

			return 0 < a && int(a) <= len(episodes)
		default:
			return s == "q"
		}
	})

	if err != nil {
		return err
	}

	switch {
	case rangeInput.MatchString(in.value):
		nums := strings.Split(in.value, " ")
		from := lo.Must(strconv.ParseInt(nums[0], 10, 16))
		to := lo.Must(strconv.ParseInt(nums[1], 10, 16))

		for i := from - 1; i < to; i++ {
			m.selectedEpisodes = append(m.selectedEpisodes, episodes[i])
		}
	case oneEpisodeInput.MatchString(in.value):
		num := lo.Must(strconv.ParseInt(in.value, 10, 16))
		m.selectedEpisodes = append(m.selectedEpisodes, episodes[num-1])
	case in.value == "q":
		m.newState(quitState)
		return nil
	}

	m.newState(episodeReadState)

	return nil
}

func (m *mini) handleEpisodeReadState() error {
	type controls struct {
		next chan struct{}
		prev chan struct{}
		stop chan struct{}
		err  chan error
	}

	var readLoop func(*source.Episode, *controls, bool, bool)

	readLoop = func(episode *source.Episode, c *controls, hasPrev, hasNext bool) {
		util.ClearScreen()
		fmt.Printf("Reading %s...\n", episode.Name)

		title(fmt.Sprintf("Currently reading %s", episode.Name))

		var options []*bind
		if hasPrev {
			options = append(options, prev)
		}
		if hasNext {
			options = append(options, next)
		}

		options = append(options, reread, back, search)

		b, _, err := menu([]fmt.Stringer{}, options...)
		if err != nil {
			c.err <- err
			return
		}

		switch b {
		case next:
			c.next <- struct{}{}
		case reread:
			readLoop(episode, c, hasPrev, hasNext)
		case back:
			m.previousState()
			c.stop <- struct{}{}
		case search:
			m.newState(animesSearchState)
			c.stop <- struct{}{}
		case quit:
			m.newState(quitState)
			c.stop <- struct{}{}
		}
	}

	c := &controls{
		next: make(chan struct{}),
		prev: make(chan struct{}),
		stop: make(chan struct{}),
		err:  make(chan error),
	}

	var i int

	for {
		var (
			hasPrev = i > 0
			hasNext = i+1 < len(m.selectedEpisodes)
		)

		go readLoop(m.selectedEpisodes[i], c, hasPrev, hasNext)

		select {
		case <-c.next:
			i++
		case <-c.prev:
			i--
		case <-c.stop:
			return nil
		case err := <-c.err:
			return err
		}
	}
}

func (m *mini) handleHistorySelectState() error {
	h, err := history.Get()
	if err != nil {
		return err
	}

	episodes := lo.Values(h)

	title("History Results >>")
	b, c, err := menu(episodes)
	if err != nil {
		return err
	}

	switch b {
	case quit:
		m.newState(quitState)
		return nil
	}

	defaultProviders := provider.Builtins()
	customProviders := provider.Customs()

	var providers = make([]*provider.Provider, 0)
	providers = append(providers, defaultProviders...)
	providers = append(providers, customProviders...)

	p, _ := lo.Find(providers, func(p *provider.Provider) bool {
		return p.ID == c.SourceID
	})

	erase := progress("Initializing Source..")
	s, err := p.CreateSource()
	if err != nil {
		return err
	}
	m.selectedSource = s
	erase()

	erase = progress("Fetching Episodes..")
	anime := &source.Anime{
		Name:   c.AnimeName,
		URL:    c.AnimeURL,
		Index:  0,
		ID:     c.AnimeID, // Fixed to map AnimeID
		Source: s,
	}
	chaps, err := m.selectedSource.EpisodesOf(anime)
	erase()

	if err != nil {
		return err
	}

	m.cachedEpisodes[anime.URL] = chaps
	m.selectedEpisodes = chaps[c.Index-1:]

	m.newState(episodeReadState)
	return nil
}
