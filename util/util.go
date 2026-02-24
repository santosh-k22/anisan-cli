// Package util provides a collection of domain-agnostic utility functions and cross-platform helpers.
package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/filesystem"
	"golang.org/x/exp/constraints"
	"golang.org/x/term"
)

// SanitizeFilename normalizes a string into a safe, cross-platform filesystem-compliant filename.
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscore
	invalid := regexp.MustCompile(`[\\/<>:;"'|?!*{}#%&^+,~\s]`)
	filename = invalid.ReplaceAllString(filename, "_")

	// Collapse multiple underscores
	collapse := regexp.MustCompile(`__+`)
	filename = collapse.ReplaceAllString(filename, "_")

	// Trim leading/trailing separators
	trim := regexp.MustCompile(`^[_\-.]+|[_\-.]+$`)
	filename = trim.ReplaceAllString(filename, "")

	return filename
}

// Quantify returns a pluralized string representation of a count and its associated labels.
func Quantify(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// Capitalize transforms the first rune of a string to its uppercase equivalent.
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// TerminalSize retrieves the current character dimensions of the terminal window.
func TerminalSize() (width, height int, err error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

// FileStem extracts the base filename from a path, excluding all file extensions.
func FileStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// ClearScreen clears the terminal buffer using the appropriate platform-specific command.
func ClearScreen() {
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	}

	switch runtime.GOOS {
	case constant.Linux, constant.Darwin:
		run("tput", "clear")
	case constant.Windows:
		run("cmd", "/c", "cls")
	}
}

// ReGroups extracts and maps named capture groups from a regular expression match.
func ReGroups(pattern *regexp.Regexp, str string) map[string]string {
	groups := make(map[string]string)
	match := pattern.FindStringSubmatch(str)
	if match == nil {
		return groups
	}

	for i, name := range pattern.SubexpNames() {
		if i > 0 && i < len(match) && name != "" {
			groups[name] = match[i]
		}
	}
	return groups
}

// PrintErasable prints an ephemeral message to the terminal and returns a closure to clear it.
func PrintErasable(msg string) (eraser func()) {
	fmt.Fprintf(os.Stdout, "\r%s", msg)
	return func() {
		fmt.Fprintf(os.Stdout, "\r%s\r", strings.Repeat(" ", len(msg)))
	}
}

// Ignore executes a function and explicitly discards its error return value.
func Ignore(f func() error) {
	_ = f()
}

// Max returns the maximum value among arguments.
func Max[T constraints.Ordered](items ...T) (max T) {
	if len(items) == 0 {
		return
	}
	max = items[0]
	for _, item := range items[1:] {
		if item > max {
			max = item
		}
	}
	return
}

// Min returns the minimum value among arguments.
func Min[T constraints.Ordered](items ...T) (min T) {
	if len(items) == 0 {
		return
	}
	min = items[0]
	for _, item := range items[1:] {
		if item < min {
			min = item
		}
	}
	return
}

// Delete recursively removes a file or directory using the virtualized filesystem API.
func Delete(path string) error {
	fs := filesystem.API()
	stat, err := fs.Stat(path)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return fs.RemoveAll(path)
	}
	return fs.Remove(path)
}
