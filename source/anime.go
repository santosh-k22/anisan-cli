// Package source defines the domain models and interfaces for media discovery and retrieval.
package source

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/util"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/viper"
)

type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// Anime represents a media entity discovered through a provider search.
type Anime struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Index  uint16 `json:"index"`
	ID     string `json:"id"`
	Source Source `json:"-"`

	Episodes []*Episode `json:"episodes"`

	// Anilist integration
	Anilist  mo.Option[*anilist.Anime] `json:"anilist"`
	Metadata Metadata                  `json:"metadata"`

	cachedTempPath string
	populated      bool
}

type Metadata struct {
	Genres      []string `json:"genres"`
	Summary     string   `json:"summary"`
	Staff       Staff    `json:"staff"`
	Cover       Cover    `json:"cover"`
	BannerImage string   `json:"bannerImage"`
	Tags        []string `json:"tags"`
	Characters  []string `json:"characters"`
	Status      string   `json:"status"`
	StartDate   Date     `json:"startDate"`
	EndDate     Date     `json:"endDate"`
	Synonyms    []string `json:"synonyms"`
	Episodes    int      `json:"episodes"`
	URLs        []string `json:"urls"`
	Score       int      `json:"score"`
	Title       string   `json:"title"` // Preferred title (English/Romaji)
}

type Staff struct {
	Story       []string `json:"story"`
	Art         []string `json:"art"`
	Translation []string `json:"translation"`
	Lettering   []string `json:"lettering"`
}

type Cover struct {
	ExtraLarge string `json:"extraLarge"`
	Large      string `json:"large"`
	Medium     string `json:"medium"`
	Color      string `json:"color"`
}

func (a *Anime) String() string {
	return a.Name
}

// Name retrieves the primary display title for the anime entity.
func (a *Anime) Dirname() string {
	return util.SanitizeFilename(a.Name)
}

// Path returns the filesystem path for the anime (cache or temp).
func (a *Anime) Path(temp bool) (string, error) {
	if temp {
		if a.cachedTempPath != "" {
			return a.cachedTempPath, nil
		}
		path := where.Temp()
		a.cachedTempPath = path
		return path, nil
	}

	path := filepath.Join(where.Cache(), a.Dirname())
	err := filesystem.API().MkdirAll(path, os.ModePerm)
	return path, err
}

// GetCover returns the best available cover image URL.
func (a *Anime) GetCover() (string, error) {
	if a.Metadata.Cover.ExtraLarge != "" {
		return a.Metadata.Cover.ExtraLarge, nil
	}
	if a.Metadata.Cover.Large != "" {
		return a.Metadata.Cover.Large, nil
	}
	if a.Metadata.Cover.Medium != "" {
		return a.Metadata.Cover.Medium, nil
	}
	return "", fmt.Errorf("no cover found")
}

// BindWithAnilist synchronizes the local anime entity with Anilist service metadata.
func (a *Anime) BindWithAnilist() error {
	if a.Anilist.IsPresent() {
		return nil
	}

	log.Infof("binding %s with anilist", a.Name)
	al, err := anilist.FindClosest(a.Name)
	if err != nil {
		log.Error(err)
		return err
	}

	a.Anilist = mo.Some(al)
	return nil
}

// PopulateMetadata retrieves and assigns extended metadata fields for the anime entity.
func (a *Anime) PopulateMetadata(progress func(string)) error {
	if a.populated {
		return nil
	}
	a.populated = true

	progress("Fetching metadata from anilist")
	log.Infof("Populating metadata for %s", a.Name)

	if err := a.BindWithAnilist(); err != nil {
		progress("Failed to fetch metadata")
		return err
	}

	al, ok := a.Anilist.Get()
	if !ok || al == nil {
		return fmt.Errorf("anime '%s' not found on Anilist", a.Name)
	}

	a.copyAnilistMetadata(al)
	return nil
}

func (a *Anime) copyAnilistMetadata(al *anilist.Anime) {
	a.Metadata.Title = al.Name()
	a.Metadata.Genres = al.Genres

	// ... (rest of the function)

	// Clean summary (remove HTML tags)
	summary := strings.ReplaceAll(al.Description, "<br>", "\n")
	re := regexp.MustCompile("<.*?>")
	a.Metadata.Summary = re.ReplaceAllString(summary, "")

	a.Metadata.Characters = make([]string, len(al.Characters.Nodes))
	for i, n := range al.Characters.Nodes {
		a.Metadata.Characters[i] = n.Name.Full
	}

	for _, tag := range al.Tags {
		if tag.Rank >= viper.GetInt(key.MetadataTagRelevanceThreshold) {
			a.Metadata.Tags = append(a.Metadata.Tags, tag.Name)
		}
	}

	a.Metadata.Cover.ExtraLarge = al.CoverImage.ExtraLarge
	a.Metadata.Cover.Large = al.CoverImage.Large
	a.Metadata.Cover.Medium = al.CoverImage.Medium
	a.Metadata.Cover.Color = al.CoverImage.Color
	a.Metadata.BannerImage = al.BannerImage

	a.Metadata.StartDate = Date(al.StartDate)
	a.Metadata.EndDate = Date(al.EndDate)
	a.Metadata.Status = strings.ReplaceAll(al.Status, "_", " ")
	a.Metadata.Synonyms = al.Synonyms
	a.Metadata.Episodes = al.Episodes
	a.Metadata.Score = al.AverageScore

	for _, staff := range al.Staff.Edges {
		role := strings.ToLower(staff.Role)
		name := staff.Node.Name.Full
		if strings.Contains(role, "story") {
			a.Metadata.Staff.Story = append(a.Metadata.Staff.Story, name)
		}
		if strings.Contains(role, "art") {
			a.Metadata.Staff.Art = append(a.Metadata.Staff.Art, name)
		}
		if strings.Contains(role, "translator") {
			a.Metadata.Staff.Translation = append(a.Metadata.Staff.Translation, name)
		}
		if strings.Contains(role, "lettering") {
			a.Metadata.Staff.Lettering = append(a.Metadata.Staff.Lettering, name)
		}
	}

	urls := []string{al.SiteURL}
	for _, e := range al.External {
		urls = append(urls, e.URL)
	}
	urls = append(urls, fmt.Sprintf("https://myanimelist.net/anime/%d", al.IDMal))
	a.Metadata.URLs = lo.Filter(urls, func(u string, _ int) bool { return u != "" })
}

// SeriesJSON returns the JSON representation of the anime.
func (a *Anime) SeriesJSON() []byte {
	b, _ := json.Marshal(a)
	return b
}
