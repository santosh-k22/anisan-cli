// Package version provides unified mechanisms for application version tracking, update discovery, and compatibility validation.
package version

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/util"
	"github.com/anisan-cli/anisan/where"
	"github.com/metafates/gache"
)

var versionCacher = gache.New[string](&gache.Options{
	Path:       filepath.Join(where.Cache(), "version.json"),
	Lifetime:   time.Hour * 24 * 2,
	FileSystem: &filesystem.GacheFs{},
})

// Latest retrieves the most recent stable application version identifier from the remote update registry.
// It queries the GitHub Releases API and caches the result for performance and rate-limit mitigation.
func Latest() (version string, err error) {
	ver, expired, err := versionCacher.Get()
	if err != nil {
		return "", err
	}

	if !expired && ver != "" {
		return ver, nil
	}

	resp, err := http.Get("https://api.github.com/repos/anisan-cli/anisan/releases/latest")
	if err != nil {
		return
	}

	defer util.Ignore(resp.Body.Close)

	var release struct {
		TagName string `json:"tag_name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return
	}

	// Sanitization: Normalize the release identifier by stripping the 'v' prefix if present.
	if release.TagName == "" {
		err = errors.New("empty tag name")
		return
	}

	version = release.TagName[1:]
	_ = versionCacher.Set(version)
	return
}
