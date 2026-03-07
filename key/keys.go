// Package key defines the canonical set of configuration identifiers used for centralized settings management.
package key

// DefinedFieldsCount represents the total cardinality of the application configuration schema.
const DefinedFieldsCount = 23

// Provider Source Identifiers - these keys manage the registration and selection of scraping providers.
const (
	DefaultSources = "sources.default"
)

// Metadata Configuration - these keys govern the retrieval and processing of media metadata.
const (
	TrackerFetchMetadata          = "tracker.fetch_metadata"
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

// Tracker Service Integration - unified keys for media tracking behavior.
const (
	TrackerEnable   = "tracker.enable"
	TrackerAutoLink = "tracker.auto_link"
)

// Tracker Authentication - unified token caches.
const (
	TrackerMalClientID  = "tracker.mal.client_id"
	TrackerMalToken     = "tracker.mal.token"
	TrackerAnilistToken = "tracker.anilist.token"
)

// Synchronization Registry - these keys determine the active media tracking and metadata backends.
const (
	TrackerBackend = "tracker.backend"
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
