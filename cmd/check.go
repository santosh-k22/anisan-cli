// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/style"
	"github.com/charmbracelet/lipgloss"
)

// CheckDependencies verifies the availability of required system dependencies.
// The current implementation validates the presence of 'mpv' in the system PATH.
func CheckDependencies() {
	_, err := exec.LookPath("mpv")
	if err != nil {
		printMissingDependencyError("mpv")
		os.Exit(1)
	}
}

func printMissingDependencyError(dep string) {
	var installCmd string
	switch runtime.GOOS {
	case "darwin":
		installCmd = "brew install mpv"
	case "linux":
		installCmd = "sudo apt install mpv" // Generic, maybe check distro
	case "windows":
		installCmd = "scoop install mpv"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.HiRed).
		Padding(1, 2).
		Margin(1, 0)

	title := style.New().Bold(true).Foreground(style.HiRed).Render(fmt.Sprintf("%s Error: Missing Dependency", icon.Get(icon.Fail)))
	body := style.New().Foreground(style.Text).Render(fmt.Sprintf("The required dependency '%s' was not found in your PATH.", dep))

	suggestion := ""
	if installCmd != "" {
		suggestion = fmt.Sprintf("\n\nTo install it, try running:\n  %s", style.New().Foreground(style.AccentColor).Bold(true).Render(installCmd))
	}

	fmt.Println(box.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"\n",
			body,
			suggestion,
		),
	))
}
