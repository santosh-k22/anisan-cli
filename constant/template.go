// Package constant defines immutable application-level identifiers and configuration defaults.
package constant

// Scraper Function Identifiers - these constants define the required global function signatures for Lua scraper modules.
const (
	SearchAnimesFn  = "SearchAnimes"
	AnimeEpisodesFn = "AnimeEpisodes"
	EpisodeVideosFn = "EpisodeVideos"
)

// SourceTemplate is a Go text/template for scaffolding new Lua scraper files.
const SourceTemplate = `{{ $divider := repeat "-" (plus (max (len .URL) (len .Name) (len .Author) 3) 12) }}{{ $divider }}
-- @name    {{ .Name }} 
-- @url     {{ .URL }}
-- @author  {{ .Author }} 
-- @license MIT
{{ $divider }}


---@alias anime { name: string, url: string, author: string|nil, genres: string|nil, summary: string|nil }
---@alias episode { name: string, url: string, volume: string|nil, anime_summary: string|nil, anime_author: string|nil, anime_genres: string|nil }


----- IMPORTS -----
--- END IMPORTS ---



----- VARIABLES -----
--- END VARIABLES ---



----- MAIN -----

--- Searches for anime with given query.
-- @param query string Query to search for
-- @return anime[] Table of animes
function {{ .SearchAnimesFn }}(query)
	return {}
end


--- Gets the list of all anime episodes.
-- @param animeURL string URL of the anime
-- @return episode[] Table of episodes
function {{ .AnimeEpisodesFn }}(animeURL)
	return {}
end


--- END MAIN ---




----- HELPERS -----
--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua
`
