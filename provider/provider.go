// Package provider manages built-in and custom scraping providers.
package provider

import (
	"bytes"
	"path/filepath"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/provider/custom"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/util"
	"github.com/anisan-cli/anisan/where"
)

// Provider represents a source provider.
type Provider struct {
	ID           string
	Name         string
	UsesHeadless bool // Indicates whether the provider requires a headless browser.
	IsCustom     bool // Reserved for Lua-based providers.
	CreateSource func() (source.Source, error)
}

func (p *Provider) String() string {
	return p.Name
}

// Builtins returns built-in providers.
func Builtins() []*Provider {
	return []*Provider{}
}

// Customs returns all available Lua providers.
func Customs() []*Provider {
	providers, _ := CustomProviders()
	return providers
}

// Get finds a provider by name.
func Get(name string) (*Provider, bool) {
	for _, p := range Customs() {
		if p.Name == name {
			return p, true
		}
	}
	return nil, false
}

func CustomProviders() ([]*Provider, error) {
	files, err := filesystem.API().ReadDir(where.Sources())
	if err != nil {
		return nil, err
	}

	var providers []*Provider
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".lua" {
			continue
		}

		if f.Name() == "common.lua" {
			continue
		}

		path := filepath.Join(where.Sources(), f.Name())
		name := util.FileStem(f.Name())

		providers = append(providers, &Provider{
			ID:           custom.IDfromName(name),
			Name:         name,
			UsesHeadless: isHeadless(path),
			IsCustom:     true,
			CreateSource: func() (source.Source, error) {
				return custom.LoadSource(path)
			},
		})
	}

	return providers, nil
}

// Helpers

func isHeadless(path string) bool {
	content, err := filesystem.API().ReadFile(path)
	if err != nil {
		return false
	}

	match := [][]byte{
		[]byte("require(\"headless\")"),
		[]byte("require('headless')"),
	}

	for _, m := range match {
		if bytes.Contains(content, m) {
			return true
		}
	}
	return false
}
