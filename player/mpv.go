package player

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anisan-cli/anisan/log"
)

const (
	socketWaitRetries = 10
	socketWaitDelay   = 300 * time.Millisecond
)

// MPV implements the Player interface using mpv's JSON-IPC protocol.
type MPV struct {
	socketPath string
	cmd        *exec.Cmd
	exited     chan struct{} // closed when mpv process exits
	tickerStop chan struct{} // signals ticker to stop
	mu         sync.Mutex    // Protects socket writes
}

// NewMPV creates a new MPV player instance (does not start playback).
func NewMPV() *MPV {
	return &MPV{
		exited: make(chan struct{}),
	}
}

// Play starts playback of the given URL. If mpv is already running,
// it loads the new file into the existing instance via IPC.
func (m *MPV) Play(rawURL string, title string, headers map[string]string) error {
	// Sanitize the URL to prevent flag injection from Lua scripts
	safeURL, err := sanitizeMediaTarget(rawURL)
	if err != nil {
		return fmt.Errorf("invalid media target: %w", err)
	}

	// Sanitize title to prevent IPC issues
	safeTitle := sanitizeTitle(title)

	// Construct header string if present
	var headerString string
	if len(headers) > 0 {
		var hBuilder strings.Builder
		for k, v := range headers {
			if hBuilder.Len() > 0 {
				hBuilder.WriteString(",")
			}
			// Replace commas in values if any (simple sanitization)
			val := strings.ReplaceAll(v, ",", "%2C")
			hBuilder.WriteString(fmt.Sprintf("%s: %s", k, val))
		}
		headerString = hBuilder.String()
	}

	// Generate a random socket path using os.TempDir() for cross-platform support
	// (macOS $TMPDIR is /var/folders/... not /tmp/)
	if m.socketPath == "" {
		randomBytes := make([]byte, 4)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("generate socket name: %w", err)
		}
		m.socketPath = filepath.Join(os.TempDir(), fmt.Sprintf("anisan-%x.sock", randomBytes))
	}

	// Build mpv arguments.
	// CRUCIAL: Pass ONLY the socket, title, and URL.
	// Do NOT pass --vo, --profile, --hwdec — respect user's mpv.conf.
	args := []string{
		"--no-terminal",
		"--really-quiet",
		fmt.Sprintf("--input-ipc-server=%s", m.socketPath),
		fmt.Sprintf("--force-media-title=%s", safeTitle),
		fmt.Sprintf("--title=%s", safeTitle), // Some mpv builds only respect --title
		"--force-window=yes",
		"--idle=yes",
	}

	if headerString != "" {
		args = append(args, fmt.Sprintf("--http-header-fields=%s", headerString))
	}

	args = append(args, safeURL)

	m.cmd = exec.Command("mpv", args...)

	// Detach from parent process group to prevent cascading shell panics.
	m.cmd.SysProcAttr = sysProcAttr()

	// Disable standard pipes to prevent resource leaks.
	m.cmd.Stdout = nil
	m.cmd.Stderr = nil
	m.cmd.Stdin = nil

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start mpv: %w", err)
	}

	// Background goroutine to reap the process and prevent zombies
	m.exited = make(chan struct{})
	go func() {
		_ = m.cmd.Wait()
		close(m.exited)
	}()

	// Wait for the IPC socket to become available
	if err := m.waitForSocket(); err != nil {
		// If socket never became ready, kill the orphaned process
		if m.cmd.Process != nil {
			select {
			case <-m.exited:
				// Already exited
			default:
				log.Warnf("killing mpv: socket never became ready")
				_ = m.cmd.Process.Kill()
			}
		}
		return fmt.Errorf("mpv socket not ready: %w", err)
	}

	return nil
}

// Wait returns a channel that is closed when the mpv process exits.
func (m *MPV) Wait() <-chan struct{} {
	return m.exited
}

// waitForSocket polls until the mpv IPC socket is accepting connections.
func (m *MPV) waitForSocket() error {
	for i := 0; i < socketWaitRetries; i++ {
		time.Sleep(socketWaitDelay)

		// Check if process already exited
		select {
		case <-m.exited:
			return fmt.Errorf("mpv exited before socket was ready")
		default:
		}

		conn, err := net.Dial("unix", m.socketPath)
		if err == nil {
			conn.Close()
			return nil
		}
	}
	return fmt.Errorf("socket %s not ready after %d attempts", m.socketPath, socketWaitRetries)
}

// GetTimePos returns the current playback position in seconds.
func (m *MPV) GetTimePos() (float64, error) {
	return m.getFloatProperty("time-pos")
}

// GetDuration returns the total duration of the current media in seconds.
func (m *MPV) GetDuration() (float64, error) {
	return m.getFloatProperty("duration")
}

// GetPercentWatched returns the percentage of the media that has been watched.
func (m *MPV) GetPercentWatched() (float64, error) {
	pos, err := m.GetTimePos()
	if err != nil {
		return 0, err
	}

	dur, err := m.GetDuration()
	if err != nil || dur <= 0 {
		return 0, err
	}

	return (pos / dur) * 100, nil
}

// GetPausedStatus returns whether playback is currently paused.
func (m *MPV) GetPausedStatus() (bool, error) {
	data, err := m.sendCommand([]interface{}{"get_property", "pause"})
	if err != nil {
		return false, err
	}
	paused, ok := data.(bool)
	if !ok {
		return false, nil
	}
	return paused, nil
}

// HasActivePlayback checks if mpv currently has active media playing.
// Returns false (not error) for "property unavailable" — nothing loaded.
func (m *MPV) HasActivePlayback() (bool, error) {
	data, err := m.sendCommand([]interface{}{"get_property", "time-pos"})
	if err != nil {
		// "property unavailable" means nothing is loaded — valid state
		if strings.Contains(err.Error(), "property unavailable") {
			return false, nil
		}
		return false, err
	}
	return data != nil, nil
}

// Seek moves playback to the given absolute position in seconds.
func (m *MPV) Seek(seconds float64) error {
	_, err := m.sendCommand([]interface{}{"seek", seconds, "absolute"})
	return err
}

// IsRunning reports whether mpv is responding to IPC commands.
func (m *MPV) IsRunning() bool {
	if m.socketPath == "" {
		return false
	}

	// Fast check: process already exited?
	select {
	case <-m.exited:
		return false
	default:
	}

	_, err := m.sendCommand([]interface{}{"get_property", "pid"})
	return err == nil
}

// StartIPCTicker starts a background ticker that polls the player for time-pos
// and calls the given callback every second.
func (m *MPV) StartIPCTicker(callback func(timePos int, duration int)) {
	if m.tickerStop != nil {
		// Ticker already running
		return
	}

	m.tickerStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-m.tickerStop:
				return
			case <-m.exited:
				// Player exited, stop ticker
				m.tickerStop = nil
				return
			case <-ticker.C:
				if !m.IsRunning() {
					continue
				}

				pos, err := m.GetTimePos()
				if err != nil {
					continue
				}

				dur, err := m.GetDuration()
				if err != nil {
					// Duration might be unknown for streams, just send 0 or keep polling
					dur = 0
				}

				callback(int(pos), int(dur))
			}
		}
	}()
}

// StopIPCTicker stops the background ticker if it's running.
func (m *MPV) StopIPCTicker() {
	if m.tickerStop != nil {
		close(m.tickerStop)
		m.tickerStop = nil
	}
}

// Close shuts down the mpv process and cleans up resources.
func (m *MPV) Close() error {
	m.StopIPCTicker()

	if m.socketPath == "" {
		return nil
	}

	// Try graceful quit via IPC
	_, _ = m.sendCommand([]interface{}{"quit"})

	// Wait for process to exit (with timeout)
	select {
	case <-m.exited:
		// Clean exit
	case <-time.After(3 * time.Second):
		// Force kill if graceful quit didn't work
		_ = killProcess(m.cmd)
	}

	// Clean up the socket file
	_ = os.Remove(m.socketPath)

	return nil
}

// Socket returns the IPC socket path.
func (m *MPV) Socket() string {
	return m.socketPath
}

// TogglePause toggles the pause state
func (m *MPV) TogglePause() error {
	return m.Set("pause", "!pause") // MPV cycle command or just set pause
}

// Set a property
func (m *MPV) Set(property string, value interface{}) error {
	_, err := m.sendCommand([]interface{}{"set_property", property, value})
	return err
}

// getFloatProperty is a helper to retrieve a float64 mpv property via IPC.
func (m *MPV) getFloatProperty(name string) (float64, error) {
	data, err := m.sendCommand([]interface{}{"get_property", name})
	if err != nil {
		return 0, err
	}

	if data == nil {
		return 0, fmt.Errorf("property %s: nil response", name)
	}

	val, ok := data.(float64)
	if !ok {
		return 0, fmt.Errorf("property %s: expected float64, got %T", name, data)
	}

	return val, nil
}

// sanitizeMediaTarget validates that a URL is safe to pass to mpv.
// Prevents flag injection from untrusted Lua scripts.
func sanitizeMediaTarget(link string) (string, error) {
	l := strings.TrimSpace(link)
	if l == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Reject control characters
	if strings.ContainsAny(l, "\x00\n\r") {
		return "", fmt.Errorf("invalid control characters in URL")
	}

	// Prevent flag injection: URLs must not start with -
	if strings.HasPrefix(l, "-") {
		return "", fmt.Errorf("url must not start with '-' (looks like a flag)")
	}

	// If it contains "://", validate as URL
	if strings.Contains(l, "://") {
		u, err := url.Parse(l)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			return l, nil
		default:
			return "", fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
		}
	}

	// Treat as local file path
	return filepath.Clean(l), nil
}

// SetChapters sets the chapters for the current media.
// This provides visual feedback in the MPV UI (timeline).
func (m *MPV) SetChapters(chapters []map[string]interface{}) error {
	_, err := m.sendCommand([]interface{}{"set_property", "chapter-list", chapters})
	return err
}

// sanitizeTitle cleanups up the title for MPV
func sanitizeTitle(title string) string {
	t := strings.ReplaceAll(title, "\n", " ")
	t = strings.ReplaceAll(t, "\r", " ")
	t = strings.ReplaceAll(t, "\t", " ")
	// Remove null bytes
	t = strings.ReplaceAll(t, "\x00", "")
	return strings.TrimSpace(t)
}
