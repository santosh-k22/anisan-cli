// Package version provides unified mechanisms for application version tracking, update discovery, and compatibility validation.
package version

import (
	"fmt"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/util"
	"github.com/spf13/viper"
)

// Notify displays a terminal alert if a more recent stable application version is available.
func Notify() {
	if !viper.GetBool(key.CliVersionCheck) {
		return
	}

	erase := util.PrintErasable(fmt.Sprintf("%s Checking if new version is available...", icon.Get(icon.Progress)))
	version, err := Latest()
	erase()
	if err == nil {
		comp, err := Compare(version, constant.Version)
		if err == nil && comp <= 0 {
			return
		}
	}

	fmt.Printf(`
%s New version is available %s %s
%s

`,
		style.Fg(color.Green)("▇▇▇"),
		style.Bold(version),
		style.Faint(fmt.Sprintf("(You're on %s)", constant.Version)),
		style.Faint("https://github.com/anisan-cli/anisan/releases/tag/v"+version),
	)

}
