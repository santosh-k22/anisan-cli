// Package open provides a cross-platform abstraction for launching files and URLs with the system's default handler.
package open

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anisan-cli/anisan/constant"
)

// Run opens the specified input (URL or file path) using the default system handler and waits for completion.
func Run(input string) error {
	cmd, ok := command(input)
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// Start opens the specified input using the default system handler asynchronously.
func Start(input string) error {
	cmd, ok := command(input)
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// RunWith opens the specified input using a specific application and waits for the process to exit.
func RunWith(input, app string) error {
	if app == "" {
		return Run(input)
	}
	cmd, ok := commandWith(input, app)
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// StartWith opens the specified input using a specific application asynchronously.
func StartWith(input, app string) error {
	if app == "" {
		return Start(input)
	}
	cmd, ok := commandWith(input, app)
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func command(input string) (*exec.Cmd, bool) {
	switch runtime.GOOS {
	case constant.Windows:
		rundll := filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")
		return exec.Command(rundll, "url.dll,FileProtocolHandler", input), true
	case constant.Darwin:
		return exec.Command("open", input), true
	case constant.Linux:
		return exec.Command("xdg-open", input), true
	case constant.Android:
		return exec.Command("termux-open", input), true
	default:
		return nil, false
	}
}

func commandWith(input, app string) (*exec.Cmd, bool) {
	switch runtime.GOOS {
	case constant.Windows:
		// Sanitization: The Windows 'start' command requires escaping the '&' character for multi-parameter URLs.
		escaped := strings.ReplaceAll(input, "&", "^&")
		return exec.Command("cmd", "/C", "start", "", app, escaped), true
	case constant.Darwin:
		return exec.Command("open", "-a", app, input), true
	case constant.Linux:
		return exec.Command(app, input), true
	case constant.Android:
		return exec.Command("termux-open", "--choose", input), true
	default:
		return nil, false
	}
}
