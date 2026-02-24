// Package key defines the canonical set of configuration identifiers used for centralized settings management.
package key

// DefinedFieldsCount represents the total cardinality of the application configuration schema.
const DefinedFieldsCount = 22

// Provider Source Identifiers - these keys manage the registration and selection of scraping providers.
const (
	DefaultSources = "sources.default"
)

// Metadata Configuration - these keys govern the retrieval and processing of media metadata.
const (
	MetadataFetchAnilist          = "metadata.fetch_anilist"
	MetadataTagRelevanceThreshold = "metadata.tag_relevance_threshold"
)

// History Tracking - these keys configure the persistence of media consumption state.
const (
	HistorySaveOnRead = "history.save_on_read"
)

// Search Interaction - these keys define the UI/UX parameters for search discovery.
const (
	SearchShowQuerySuggestions = "search.show_query_suggestions"
)

// Minimalist (Mini) Mode - these keys configure the specialized lightweight TUI.
const (
	MiniSearchLimit = "mini.search_limit"
)

// Iconography - these keys manage the visual rendering of UI symbols.
const (
	IconsVariant = "icons.variant"
)

// Anilist Service Integration - these keys manage the authentication and synchronization with Anilist.
const (
	AnilistEnable            = "anilist.enable"
	AnilistID                = "anilist.id"
	AnilistSecret            = "anilist.secret"
	AnilistCode              = "anilist.code"
	AnilistLinkOnAnimeSelect = "anilist.link_on_anime_select"
)

// Terminal User Interface (TUI) - these keys define the primary interactive environment's styling and logic.
const (
	TUIItemSpacing        = "tui.item_spacing"
	TUIReadOnEnter        = "tui.read_on_enter"
	TUISearchPromptString = "tui.search_prompt"
	TUIShowURLs           = "tui.show_urls"
	TUIReverseEpisodes    = "tui.reverse_episodes"
)

// Media Playback - these keys maintain the state and configuration for external video players.
const (
	PlayerCompletionPercentage = "player.completion_percentage"
)

// Logging Infrastructure - these keys manage the application's internal diagnostics and auditing system.
const (
	LogsWrite = "logs.write"
	LogsLevel = "logs.level"
	LogsJson  = "logs.json"
)

// CLI Execution Environment - these flags and settings govern the non-TUI application behavior.
const (
	CliColored      = "cli.colored"
	CliVersionCheck = "cli.version_check"
)

// Advanced Playback Parameters - these keys manage integration with specialized playback services like AniSkip.
const (
	Aniskip = "player.aniskip"
	Player  = "player.default"
)
