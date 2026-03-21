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

var (
	invalidFilenameCharsRegex = regexp.MustCompile(`[\\/<>:;"'|?!*{}#%&^+,~\s]`)
	collapseUnderscoresRegex  = regexp.MustCompile(`__+`)
	trimFilenameEdgesRegex    = regexp.MustCompile(`^[_\-.]+|[_\-.]+$`)
)

// SanitizeFilename normalizes a string into a safe, cross-platform filesystem-compliant filename.
func SanitizeFilename(filename string) string {
	filename = invalidFilenameCharsRegex.ReplaceAllString(filename, "_")
	filename = collapseUnderscoresRegex.ReplaceAllString(filename, "_")
	filename = trimFilenameEdgesRegex.ReplaceAllString(filename, "")
	return filename
}

func Quantify(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func TerminalSize() (width, height int, err error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

func FileStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

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

func PrintErasable(msg string) (eraser func()) {
	fmt.Fprintf(os.Stdout, "\r%s", msg)
	return func() {
		fmt.Fprintf(os.Stdout, "\r%s\r", strings.Repeat(" ", len(msg)))
	}
}

func Ignore(f func() error) {
	_ = f()
}

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

func Delete(path string) error {
	return filesystem.API().RemoveAll(path)
}
