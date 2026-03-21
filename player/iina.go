// Package player defines a unified abstraction layer for media playback engines.
package player

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/anisan-cli/anisan/internal/tracker"
	"github.com/anisan-cli/anisan/key"
	"github.com/spf13/viper"
)

// IINA implements the Player interface for macOS native IINA playback.
type IINA struct {
	cmd    *exec.Cmd
	exited chan struct{}
}

func NewIINA() *IINA {
	return &IINA{
		exited: make(chan struct{}),
	}
}

// SetTrackerContext implements the Player interface for tracking metadata.
func (m *IINA) SetTrackerContext(t tracker.MediaTracker, id, ep, total int, guard *atomic.Bool) {
	// IINA native playback via 'open' does not support background IPC synchronization.
}

func (m *IINA) Play(rawURL string, title string, headers map[string]string) error {
	args, err := m.buildArgs(rawURL, title, headers)
	if err != nil {
		return err
	}

	m.cmd = exec.Command("open", args...)

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("LaunchServices failed to invoke IINA: %w", err)
	}

	// Wait for process to detach/finish
	go func() {
		_ = m.cmd.Wait()
		close(m.exited)
	}()

	return nil
}

// PlaySync starts playback synchronously and blocks until the player process exits.
func (m *IINA) PlaySync(rawURL string, title string, headers map[string]string) error {
	args, err := m.buildArgs(rawURL, title, headers)
	if err != nil {
		return err
	}

	// Add the -W flag to wait for the application to exit.
	args = append([]string{"-W"}, args...)

	m.cmd = exec.Command("open", args...)
	return m.cmd.Run()
}

func (m *IINA) buildArgs(rawURL string, title string, headers map[string]string) ([]string, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("IINA is only supported on macOS")
	}

	args := []string{"-a", "IINA"}

	// IINA accepts mpv-specific arguments via the '--args' flag separator.
	if viper.GetBool(key.Aniskip) {
		args = append(args, "--args", fmt.Sprintf("--mpv-force-media-title=%s", title))
	} else {
		args = append(args, "--args", fmt.Sprintf("--force-media-title=%s", title))
	}

	// Headers
	if len(headers) > 0 {
		var hBuilder strings.Builder
		for k, v := range headers {
			if hBuilder.Len() > 0 {
				hBuilder.WriteString(",")
			}
			hBuilder.WriteString(fmt.Sprintf("%s: %s", k, v))
		}
		args = append(args, fmt.Sprintf("--http-header-fields=%s", hBuilder.String()))
	}

	args = append(args, rawURL)
	return args, nil
}

func (m *IINA) Wait() <-chan struct{} {
	return m.exited
}

// Stub implementations for the rest of the interface
func (m *IINA) TogglePause() error                  { return nil }
func (m *IINA) GetTimePos() (float64, error)        { return 0, fmt.Errorf("not supported on IINA") }
func (m *IINA) GetDuration() (float64, error)       { return 0, fmt.Errorf("not supported on IINA") }
func (m *IINA) GetPercentWatched() (float64, error) { return 0, fmt.Errorf("not supported on IINA") }
func (m *IINA) GetPausedStatus() (bool, error)      { return false, fmt.Errorf("not supported on IINA") }
func (m *IINA) HasActivePlayback() (bool, error)    { return false, nil }
func (m *IINA) Seek(seconds float64) error          { return nil }
func (m *IINA) IsRunning() bool {
	select {
	case <-m.exited:
		return false
	default:
		return true
	}
}
func (m *IINA) Close() error {
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
	return nil
}
func (m *IINA) Socket() string                                          { return "iina-native" }
func (m *IINA) StartIPCTicker(callback func(timePos int, duration int)) {}
func (m *IINA) StopIPCTicker()                                          {}
