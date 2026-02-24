// Package anilist provides a client for the Anilist GraphQL API.
package anilist

// date represents a calendar date in the Anilist GraphQL API.
type date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type Anime struct {
	// Title is the structured title metadata for the anime.
	Title struct {
		// Romaji is the romanized title of the anime.
		Romaji string `json:"romaji" jsonschema:"description=Romanized title of the anime."`
		// English is the english title of the anime.
		English string `json:"english" jsonschema:"description=English title of the anime."`
		// Native is the native title of the anime. (Usually in kanji)
		Native string `json:"native" jsonschema:"description=Native title of the anime. Usually in kanji."`
	} `json:"title"`
	// ID is the unique identifier for the anime on Anilist.
	ID int `json:"id" jsonschema:"description=ID of the anime on Anilist."`
	// Description is the plot summary or description of the anime in HTML format.
	Description string `json:"description" jsonschema:"description=Description of the anime in html format."`
	// CoverImage contains URLs for different sizes of the anime's cover art.
	CoverImage struct {
		// ExtraLarge is the url of the extra large cover image.
		// If the image is not available, large will be used instead.
		ExtraLarge string `json:"extraLarge" jsonschema:"description=URL of the extra large cover image. If the image is not available, large will be used instead."`
		// Large is the url of the large cover image.
		Large string `json:"large" jsonschema:"description=URL of the large cover image."`
		// Medium is the url of the medium cover image.
		Medium string `json:"medium" jsonschema:"description=URL of the medium cover image."`
		// Color is the average color of the cover image.
		Color string `json:"color" jsonschema:"description=Average color of the cover image."`
	} `json:"coverImage" jsonschema:"description=Cover image of the anime."`
	// BannerImage is the URL for the anime's large banner image.
	BannerImage string `json:"bannerImage" jsonschema:"description=Banner image of the anime."`
	// Tags are metadata tags associated with the anime.
	Tags []struct {
		// Name of the tag.
		Name string `json:"name" jsonschema:"description=Name of the tag."`
		// Description of the tag.
		Description string `json:"description" jsonschema:"description=Description of the tag."`
		// Rank of the tag. How relevant it is to the anime from 1 to 100.
		Rank int `json:"rank" jsonschema:"description=Rank of the tag. How relevant it is to the anime from 1 to 100."`
	} `json:"tags"`
	// Genres is a collection of strings representing the anime's genres.
	Genres []string `json:"genres" jsonschema:"description=Genres of the anime."`
	// Characters lists the primary characters featured in the anime.
	Characters struct {
		Nodes []struct {
			Name struct {
				// Full is the full name of the character.
				Full string `json:"full" jsonschema:"description=Full name of the character."`
				// Native is the native name of the character. Usually in kanji.
				Native string `json:"native" jsonschema:"description=Native name of the character. Usually in kanji."`
			} `json:"name"`
		} `json:"nodes"`
	} `json:"characters"`
	// Staff lists the production staff members associated with the anime.
	Staff struct {
		Edges []struct {
			// Role is the primary responsibility of the staff member on this project.
			Role string `json:"role" jsonschema:"description=Role of the staff member."`
			Node struct {
				Name struct {
					// Full is the full name of the staff member.
					Full string `json:"full" jsonschema:"description=Full name of the staff member."`
				} `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"staff"`
	// StartDate is the date the anime started publishing.
	StartDate date `json:"startDate" jsonschema:"description=Date the anime started publishing."`
	// EndDate is the date the anime ended publishing.
	EndDate date `json:"endDate" jsonschema:"description=Date the anime ended publishing."`
	// Synonyms are the synonyms of the anime (Alternative titles).
	Synonyms []string `json:"synonyms" jsonschema:"description=Synonyms of the anime (Alternative titles)."`
	// Status is the status of the anime. (FINISHED, RELEASING, NOT_YET_RELEASED, CANCELLED)
	Status string `json:"status" jsonschema:"enum=FINISHED,enum=RELEASING,enum=NOT_YET_RELEASED,enum=CANCELLED,enum=HIATUS"`
	// IDMal is the id of the anime on MyAnimeList.
	IDMal int `json:"idMal" jsonschema:"description=ID of the anime on MyAnimeList."`
	// Episodes is the total episode count from the Anilist API (used for progress tracking).
	Episodes int `json:"episodes" jsonschema:"description=Total number of episodes the anime has when complete."`
	// SiteURL is the url of the anime on Anilist.
	SiteURL string `json:"siteUrl" jsonschema:"description=URL of the anime on Anilist."`
	// Country of origin of the anime.
	Country string `json:"countryOfOrigin" jsonschema:"description=Country of origin of the anime."`
	// External urls related to the anime.
	External []struct {
		URL string `json:"url" jsonschema:"description=URL of the external link."`
	} `json:"externalLinks" jsonschema:"description=External links related to the anime."`
	// AverageScore is the average score of the anime on Anilist.
	AverageScore int `json:"averageScore" jsonschema:"description=Average score of the anime on Anilist."`
}

// Name returns the primary display name of the anime. If English is available, it is preferred; otherwise, Romaji is used.
func (m *Anime) Name() string {
	if m.Title.English == "" {
		return m.Title.Romaji
	}

	return m.Title.English
}
