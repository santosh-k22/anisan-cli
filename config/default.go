// Package config provides centralized management for application settings, defaults, and the Viper-based configuration engine.
package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/style"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

// Field represents a configuration field definition.
type Field struct {
	Key         string
	Value       any
	Description string
}

// Pretty returns a colored string representation of the field for display.
func (f *Field) Pretty() string {
	var b strings.Builder
	lo.Must0(prettyTemplate.Execute(&b, f))
	return b.String()
}

// Env returns the environment variable name for this field.
func (f *Field) Env() string {
	env := strings.ToUpper(EnvKeyReplacer.Replace(f.Key))
	prefix := strings.ToUpper(constant.Anisan + "_")
	if strings.HasPrefix(env, prefix) {
		return env
	}
	return prefix + env
}

// MarshalJSON customizes JSON output to include current and default values.
func (f *Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key         string `json:"key"`
		Value       any    `json:"value"`
		Default     any    `json:"default"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}{
		Key:         f.Key,
		Value:       viper.Get(f.Key),
		Default:     f.Value,
		Description: f.Description,
		Type:        f.typeName(),
	})
}

// typeName returns the string representation of the field's underlying value type.
func (f *Field) typeName() string {
	switch f.Value.(type) {
	case string:
		return "string"
	case int:
		return "int"
	case bool:
		return "bool"
	case []string:
		return "[]string"
	case []int:
		return "[]int"
	default:
		return "unknown"
	}
}

// Default holds the map of all configuration fields.
var Default = make(map[string]Field)

// EnvExposed holds keys that are bound to environment variables.
var EnvExposed []string

func init() {
	// Register all defaults.
	// We no longer panic on count mismatch, trusting the list below.
	// register validates and adds a new configuration field to the global registry.
	register := func(k string, v any, desc string) {
		if _, exists := Default[k]; exists {
			panic("Duplicate config key: " + k)
		}
		f := Field{Key: k, Value: v, Description: desc}
		Default[k] = f
		EnvExposed = append(EnvExposed, k)
	}

	register(key.DefaultSources, []string{"allanime"}, "Default sources to use.\nWill prompt if not set.\nType \"anisan sources list\" to show available sources")
	register(key.MetadataFetchAnilist, true, "Fetch metadata from Anilist\nIt will also cache the results to not spam the API")
	register(key.MetadataTagRelevanceThreshold, 60, "Minimum relevance of a tag to be included. From 0 to 100")
	register(key.MiniSearchLimit, 20, "Limit of search results to show")
	register(key.IconsVariant, "plain", "Icons variant.\nAvailable options are: emoji, kaomoji, plain, squares, nerd (nerd-font required)")
	register(key.HistorySaveOnRead, true, "Save history on episode watch")
	register(key.SearchShowQuerySuggestions, true, "Show query suggestions when searching")
	register(key.LogsWrite, false, "Write logs")
	register(key.LogsLevel, "info", "Available options are: (from less to most verbose)\npanic, fatal, error, warn, info, debug, trace")
	register(key.LogsJson, false, "Use json format for logs")
	register(key.AnilistEnable, false, "Enable Anilist integration")
	register(key.AnilistCode, "", "Anilist code to use for authentication")
	register(key.AnilistID, "", "Anilist ID to use for authentication")
	register(key.AnilistSecret, "", "Anilist secret to use for authentication")
	register(key.AnilistLinkOnAnimeSelect, true, "Show link to Anilist on anime select")
	register(key.TUIItemSpacing, 1, "Spacing between items in the TUI")
	register(key.TUIReadOnEnter, true, "Play episode on enter if other episodes aren't selected")
	register(key.TUISearchPromptString, "> ", "Search prompt string to use")
	register(key.TUIShowURLs, true, "Show URLs under list items")
	register(key.TUIReverseEpisodes, false, "Reverse episodes order")
	register(key.CliColored, true, "Enable colored CLI output")
	register(key.CliVersionCheck, true, "Enable automatic version check")
	register(key.Aniskip, true, "Enable automatic introduction skipping (aniskip)")
	register(key.Player, "mpv", "Media player to use (e.g., mpv, iina)")
	register(key.PlayerCompletionPercentage, 80, "Percentage required to mark an episode as watched (1-100)")
}

var prettyTemplate = lo.Must(template.New("pretty").Funcs(template.FuncMap{
	"faint":    style.Faint,
	"bold":     style.Bold,
	"purple":   style.Fg(color.Purple),
	"blue":     style.Fg(color.Blue),
	"cyan":     style.Fg(color.Cyan),
	"value":    func(k string) any { return viper.Get(k) },
	"typename": func(v any) string { return reflect.TypeOf(v).String() },
	"hl": func(v any) string {
		switch value := v.(type) {
		case bool:
			b := strconv.FormatBool(value)
			if value {
				return style.Fg(color.Green)(b)
			}
			return style.Fg(color.Red)(b)
		case string:
			return style.Fg(color.Yellow)(value)
		default:
			return fmt.Sprint(value)
		}
	},
}).Parse(`{{ faint .Description }}
{{ blue "Key:" }}     {{ purple .Key }}
{{ blue "Env:" }}     {{ .Env }}
{{ blue "Value:" }}   {{ hl (value .Key) }}
{{ blue "Default:" }} {{ hl (.Value) }}
{{ blue "Type:" }}    {{ typename .Value }}`))
