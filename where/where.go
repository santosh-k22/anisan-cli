// Package where implements a cross-platform resolver for application-specific filesystem paths.
package where

import (
	"os"
	"path/filepath"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/samber/lo"
)

// EnvConfigPath is the environment variable identifier used to override the default configuration directory.
const EnvConfigPath = "ANISAN_CONFIG_PATH"

// ensureDir guarantees the existence of a directory at the specified path, creating it if necessary.
func ensureDir(path string) string {
	lo.Must0(filesystem.API().MkdirAll(path, os.ModePerm))
	return path
}

// Config resolves the absolute path to the primary application configuration directory.
// It prioritizes the XDG_CONFIG_HOME specification on Linux and equivalent user profile paths on Darwin and Windows.
// Direct override: The path resolution can be explicitly specified via the ANISAN_CONFIG_PATH environment variable.
func Config() string {
	if custom, ok := os.LookupEnv(EnvConfigPath); ok {
		return ensureDir(custom)
	}

	base := lo.Must(os.UserConfigDir())
	return ensureDir(filepath.Join(base, constant.Anisan))
}

// Cache resolves the absolute path to the application's persistent cache directory.
// Compliance: Adheres to the XDG_CACHE_HOME specification or platform-specific equivalent.
func Cache() string {
	base, err := os.UserCacheDir()
	if err != nil {
		// Fallback: Revert to a localized cache directory if the system-provided path is inaccessible.
		base = filepath.Join(".", "cache")
	}
	return ensureDir(filepath.Join(base, constant.Anisan))
}

// Logs resolves the absolute path to the directory used for application diagnostic and audit logs.
func Logs() string {
	return ensureDir(filepath.Join(Config(), "logs"))
}

// Sources resolves the absolute path to the directory containing local provider scripts and custom scrapers.
func Sources() string {
	return ensureDir(filepath.Join(Config(), "sources"))
}

// History resolves the absolute path to the localized watch history persistence file.
func History() string {
	return filepath.Join(Config(), "history.json")
}

// AnilistBinds resolves the absolute path to the localized Anilist media mapping registry.
func AnilistBinds() string {
	return filepath.Join(Config(), "anilist.json")
}

// Queries resolves the absolute path to the localized search query suggestion registry.
func Queries() string {
	return filepath.Join(Cache(), "queries.json")
}

// Temp resolves a unique, volatile filesystem path for transient application artifacts.
func Temp() string {
	return ensureDir(filepath.Join(os.TempDir(), constant.Anisan))
}
