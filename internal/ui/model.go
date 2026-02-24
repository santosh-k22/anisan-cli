// Package ui provides internal state management and rendering utilities for ephemeral terminal notifications.
package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Model encapsulates the state for displaying non-blocking terminal alerts.
type Model struct {
	notification string
	notifiedAt   time.Time
}

// ClearNotificationMsg is a Bubbletea message used to reset the visual notification state.
type ClearNotificationMsg struct{}

// NotifySyncFailure returns a tea.Cmd to trigger a synchronization failure alert.
func NotifySyncFailure() tea.Cmd {
	return func() tea.Msg {
		return "Sync Failed - Queued for background reconciliation"
	}
}

// ClearNotification returns a delayed tea.Cmd that clears the current notification after a fixed duration.
func ClearNotification() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return ClearNotificationMsg{}
	})
}

// Update processes incoming messages to modify the notification state.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case string:
		m.notification = msg
		m.notifiedAt = time.Now()
		return ClearNotification()
	case ClearNotificationMsg:
		m.notification = ""
		return nil
	}
	return nil
}

// View injects the current notification message into the terminal view buffer.
func (m *Model) View(mainContent string) string {
	if m.notification == "" {
		return mainContent
	}

	// Standardize on a low-intensity ANSI escape sequence to minimize visual noise.
	lines := strings.Split(mainContent, "\n")
	notifier := "\033[90m" + m.notification + "\033[0m"

	if len(lines) > 0 {
		lines[len(lines)-1] = lines[len(lines)-1] + "  " + notifier
	}
	return strings.Join(lines, "\n")
}
