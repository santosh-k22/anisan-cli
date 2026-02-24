// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"fmt"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/internal/scraper"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/util"
	libs "github.com/metafates/mangal-lua-libs"
	lua "github.com/yuin/gopher-lua"
)

// IDfromName generates a canonical provider identifier for a given Lua script basename.
func IDfromName(name string) string {
	return name + " custom"
}

// LoadSource initializes a new source.Source instance by executing and validating a Lua scraper script.
func LoadSource(path string) (source.Source, error) {
	state := lua.NewState()
	libs.Preload(state)
	registerTLSClient(state) // Injected from wrapper_tls.go

	// Load and compile the Lua script (using cache if available).
	err := scraper.PreCompileAndLoad(state, path)
	if err != nil {
		return nil, err
	}

	name := util.FileStem(path)

	// Validation
	required := []string{
		constant.SearchAnimesFn,
		constant.AnimeEpisodesFn,
		constant.EpisodeVideosFn,
	}

	for _, fn := range required {
		if state.GetGlobal(fn).Type() != lua.LTFunction {
			return nil, fmt.Errorf("function %s is required but not defined in %s", fn, name)
		}
	}

	return newLuaSource(name, state)
}
