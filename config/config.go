package config

import (
	"strings"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/where"
	"github.com/spf13/viper"
)

var EnvKeyReplacer = strings.NewReplacer(".", "_")

func Setup() error {
	viper.SetConfigName(constant.Anisan)
	viper.SetConfigType("toml")
	viper.SetFs(filesystem.API())
	viper.AddConfigPath(where.Config())

	viper.SetEnvPrefix(constant.Anisan)
	viper.SetEnvKeyReplacer(EnvKeyReplacer)
	for _, env := range EnvExposed {
		viper.MustBindEnv(env)
	}

	SetupTrackerDefaults()

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

func SetupTrackerDefaults() {
	viper.SetDefault("tracker.backend", "anilist") // Options: "anilist" | "mal" | "none"
	viper.SetDefault("tracker.enable", false)
	viper.SetDefault("tracker.auto_link", true)
}
