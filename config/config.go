// Package config provides centralized management for application settings, defaults, and the Viper-based configuration engine.
package config

import (
	"strings"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/where"
	"github.com/spf13/viper"
)

// EnvKeyReplacer is a strings.Replacer used to normalize configuration keys into environment variable naming conventions.
var EnvKeyReplacer = strings.NewReplacer(".", "_")

// Setup initializes the global configuration state, including defaults, environment bindings, and localized file resolution.
func Setup() error {
	viper.SetConfigName(constant.Anisan)
	viper.SetConfigType("toml")
	viper.SetFs(filesystem.API())
	viper.AddConfigPath(where.Config())

	// Synchronize environment variable bindings.
	viper.SetEnvPrefix(constant.Anisan)
	viper.SetEnvKeyReplacer(EnvKeyReplacer)
	for _, env := range EnvExposed {
		viper.MustBindEnv(env)
	}

	// Initialize factory default values.
	viper.SetTypeByDefaultValue(true)
	for name, field := range Default {
		viper.SetDefault(name, field.Value)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return err
	}

	return nil
}
