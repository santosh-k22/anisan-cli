// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"github.com/anisan-cli/anisan/provider/custom"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolP("lenient", "l", false, "Suppress warnings regarding missing Lua metadata functions")
}

// runCmd facilitates the execution of local Lua source files for development and debugging.
var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Execute a local Lua source file",
	Long: `Initialize the Lua 5.1 virtual machine to execute a specified script. Useful for scraper development and debugging.
Optionally utilizes the internal environment as a standalone Lua interpreter.`,
	Args:    cobra.ExactArgs(1),
	Example: "  anisan run ./test.lua",
	Run: func(cmd *cobra.Command, args []string) {
		sourcePath := args[0]

		// Invoke the Lua interpreter to load and execute the target script.
		_, err := custom.LoadSource(sourcePath)
		handleErr(err)
	},
}
