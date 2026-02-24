// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/anisan-cli/anisan/color"

	"github.com/anisan-cli/anisan/config"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/where"
	levenshtein "github.com/ka-weihe/fast-levenshtein"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func errUnknownKey(key string) error {
	closest := lo.MinBy(lo.Keys(config.Default), func(a string, b string) bool {
		return levenshtein.Distance(key, a) < levenshtein.Distance(key, b)
	})
	msg := fmt.Sprintf(
		"unknown key %s, did you mean %s?",
		style.Fg(color.Red)(key),
		style.Fg(color.Yellow)(closest),
	)

	return errors.New(msg)
}

func completionConfigKeys(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return lo.Keys(config.Default), cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// configCmd serves as the parent command for managing application configuration.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration settings and defaults",
}

func init() {
	configCmd.AddCommand(configInfoCmd)
	configInfoCmd.Flags().StringSliceP("key", "k", []string{}, "Specify the configuration keys to retrieve information for")
	configInfoCmd.Flags().BoolP("json", "j", false, "Format the output as a JSON string")
	_ = configInfoCmd.RegisterFlagCompletionFunc("key", completionConfigKeys)

	configInfoCmd.SetOut(os.Stdout)
}

// configInfoCmd displays metadata and descriptions for configuration fields.
var configInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display detailed information and descriptions for specified configuration fields",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			keys   = lo.Must(cmd.Flags().GetStringSlice("key"))
			asJson = lo.Must(cmd.Flags().GetBool("json"))
			fields = lo.Values(config.Default)
		)

		if len(keys) > 0 {
			fields = make([]config.Field, 0, len(keys))

			for _, key := range keys {
				if _, ok := config.Default[key]; !ok {
					handleErr(errUnknownKey(key))
				}

				fields = append(fields, config.Default[key])
			}
		}

		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Key < fields[j].Key
		})

		if asJson {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			lo.Must0(encoder.Encode(fields))
			return
		}

		for i, field := range fields {
			fmt.Print(field.Pretty())

			if i < len(fields)-1 {
				fmt.Println()
				fmt.Println()
			}
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().StringSliceP("value", "v", []string{}, "The new value to assign to the configuration key")

	// Deprecated flags retained for backward compatibility.
	configSetCmd.Flags().BoolP("bool", "b", false, "Explicitly interpret the value as a boolean (Legacy)")
	configSetCmd.Flags().IntP("int", "i", 0, "Explicitly interpret the value as an integer (Legacy)")
}

// configSetCmd updates the value of a specific configuration key.
var configSetCmd = &cobra.Command{
	Use:               "set [key] [value]",
	Short:             "Update the value of a specified configuration key",
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completionConfigKeys,
	Run: func(cmd *cobra.Command, args []string) {
		var key string
		var value []string

		flagKey, _ := cmd.Flags().GetString("key")
		flagValue, _ := cmd.Flags().GetStringSlice("value")

		if len(args) >= 1 {
			key = args[0]
		} else if flagKey != "" {
			key = flagKey
		} else {
			handleErr(errors.New("key is required as an argument or --key flag"))
		}

		if len(args) >= 2 {
			value = args[1:]
		} else if len(flagValue) > 0 {
			value = flagValue
		} else {
			handleErr(errors.New("value is required as an argument or --value flag"))
		}

		if _, ok := config.Default[key]; !ok {
			handleErr(errUnknownKey(key))
		}

		var v any
		switch config.Default[key].Value.(type) {
		case string:
			v = value[0]
		case int:
			parsedInt, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				handleErr(fmt.Errorf("invalid integer value: %s", value))
			}

			v = int(parsedInt)
		case bool:
			parsedBool, err := strconv.ParseBool(value[0])
			if err != nil {
				handleErr(fmt.Errorf("invalid boolean value: %s", value))
			}

			v = parsedBool
		case []string:
			v = value
		}

		viper.Set(key, v)
		switch err := viper.WriteConfig(); err.(type) {
		case viper.ConfigFileNotFoundError:
			handleErr(viper.SafeWriteConfig())
		default:
			handleErr(err)
		}

		fmt.Printf(
			"%s set %s to %s\n",
			style.Fg(color.Green)(icon.Get(icon.Success)),
			style.Fg(color.Purple)(key),
			style.Fg(color.Yellow)(fmt.Sprintf("%v", v)),
		)
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configGetCmd.Flags().StringP("key", "k", "", "The specific configuration key to retrieve")
	_ = configGetCmd.RegisterFlagCompletionFunc("key", completionConfigKeys)
}

// configGetCmd retrieves the current value of a configuration key.
var configGetCmd = &cobra.Command{
	Use:               "get [key]",
	Short:             "Retrieve the current value of a specified configuration key",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completionConfigKeys,
	Run: func(cmd *cobra.Command, args []string) {
		var key string
		flagKey, _ := cmd.Flags().GetString("key")

		if len(args) >= 1 {
			key = args[0]
		} else if flagKey != "" {
			key = flagKey
		} else {
			handleErr(errors.New("key is required as an argument or --key flag"))
		}

		if _, ok := config.Default[key]; !ok {
			handleErr(errUnknownKey(key))
		}

		fmt.Println(viper.Get(key))
	},
}

func init() {
	configCmd.AddCommand(configWriteCmd)
	configWriteCmd.Flags().BoolP("force", "f", false, "Forcefully overwrite the existing configuration file")
}

// configWriteCmd serializes the current in-memory configuration to disk.
var configWriteCmd = &cobra.Command{
	Use:   "write",
	Short: "Persist the current in-memory configuration to the localized config file",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			force          = lo.Must(cmd.Flags().GetBool("force"))
			configFilePath = filepath.Join(
				where.Config(),
				fmt.Sprintf("%s.%s", constant.Anisan, "toml"),
			)
		)

		if force {
			err := filesystem.
				API().
				Remove(configFilePath)

			handleErr(err)
		}

		handleErr(viper.SafeWriteConfig())
		fmt.Printf(
			"%s wrote config to %s\n",
			style.Fg(color.Green)(icon.Get(icon.Success)),
			configFilePath,
		)
	},
}

func init() {
	configCmd.AddCommand(configDeleteCmd)
}

// configDeleteCmd removes the configuration file from the localized storage.
var configDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Permanently remove the localized configuration file from the system",
	Aliases: []string{"remove"},
	Run: func(cmd *cobra.Command, args []string) {
		err := filesystem.
			API().
			Remove(
				filepath.Join(
					where.Config(),
					fmt.Sprintf("%s.%s", constant.Anisan, "toml"),
				),
			)

		handleErr(err)
		fmt.Printf(
			"%s deleted config\n",
			style.Fg(color.Green)(icon.Get(icon.Success)),
		)
	},
}

func init() {
	configCmd.AddCommand(configResetCmd)

	configResetCmd.Flags().StringP("key", "k", "", "The configuration key to restore to its default value")
	configResetCmd.Flags().BoolP("all", "a", false, "Restore all configuration settings to their factory defaults")
	configResetCmd.MarkFlagsMutuallyExclusive("key", "all")
	_ = configResetCmd.RegisterFlagCompletionFunc("key", completionConfigKeys)
}

// configResetCmd restores configuration keys to their factory default values.
var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Restore a specified configuration key to its default value",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("key") && !cmd.Flags().Changed("all") {
			handleErr(fmt.Errorf("either --key or --all must be set"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			key = lo.Must(cmd.Flags().GetString("key"))
			all = lo.Must(cmd.Flags().GetBool("all"))
		)

		if all {
			for key, field := range config.Default {
				viper.Set(key, field.Value)
			}
		} else if _, ok := config.Default[key]; !ok {
			handleErr(errUnknownKey(key))
		} else {
			viper.Set(key, config.Default[key].Value)
		}

		switch err := viper.WriteConfig(); err.(type) {
		case viper.ConfigFileNotFoundError:
			handleErr(viper.SafeWriteConfig())
		default:
			handleErr(err)
		}

		if all {
			fmt.Printf(
				"%s reset all config values\n",
				style.Fg(color.Green)(icon.Get(icon.Success)),
			)
		} else {
			fmt.Printf(
				"%s reset %s to default value %s\n",
				style.Fg(color.Green)(icon.Get(icon.Success)),
				style.Fg(color.Purple)(key),
				style.Fg(color.Yellow)(fmt.Sprintf("%v", config.Default[key].Value)),
			)
		}
	},
}
