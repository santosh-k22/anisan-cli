// Package player defines a unified abstraction layer for media playback engines.
package player

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/anisan-cli/anisan/key"
	"github.com/spf13/viper"
)

// IINA implements the Player interface for macOS native IINA playback.
// It acts as a stub for IPC functionality since IINA does not expose
// the same IPC socket interface as mpv.
type IINA struct {
	cmd    *exec.Cmd
	exited chan struct{}
}

func NewIINA() *IINA {
	return &IINA{
		exited: make(chan struct{}),
	}
}

func (m *IINA) Play(rawURL string, title string, headers map[string]string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("IINA is only supported on macOS")
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
		var hBuilder string
		for k, v := range headers {
			if len(hBuilder) > 0 {
				hBuilder += ","
			}
			hBuilder += fmt.Sprintf("%s: %s", k, v)
		}
		args = append(args, fmt.Sprintf("--http-header-fields=%s", hBuilder))
	}

	args = append(args, rawURL)

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
