// Package player defines a unified abstraction layer for media playback engines.
package player

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/anisan-cli/anisan/log"
)

// EventCallback is the function signature for mpv event notifications.
type EventCallback func(property string, data interface{})

// EventListener provides real-time mpv event monitoring via observe_property.
type EventListener struct {
	socketPath string
	conn       net.Conn
	callback   EventCallback
	stopCh     chan struct{}
	mu         sync.Mutex
	listening  bool
}

// NewEventListener creates a new event listener for the given socket.
func NewEventListener(socketPath string, callback EventCallback) *EventListener {
	return &EventListener{
		socketPath: socketPath,
		callback:   callback,
		stopCh:     make(chan struct{}),
	}
}

// Start begins listening for mpv property change events.
// It sets up property observers and starts a dedicated read loop.
func (el *EventListener) Start() error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.listening {
		return nil
	}

	// Subscribe to property change events via IPC
	// observe_property <id> <property> — mpv sends notifications when they change
	properties := []struct {
		id   int
		name string
	}{
		{1, "time-pos"},    // For skip detection + sync progress
		{2, "pause"},       // For pausing sync timer
		{3, "seeking"},     // For seek detection
		{4, "eof-reached"}, // For episode completion detection
	}

	for _, prop := range properties {
		_, err := doSendCommand(el.socketPath, []interface{}{"observe_property", prop.id, prop.name})
		if err != nil {
			return fmt.Errorf("observe %s: %w", prop.name, err)
		}
	}

	// Open a persistent connection for the event read loop
	conn, err := net.Dial("unix", el.socketPath)
	if err != nil {
		return fmt.Errorf("event listener connect: %w", err)
	}
	el.conn = conn
	el.listening = true

	// Start the event read loop in a background goroutine
	go el.readLoop()

	log.Infof("mpv event listener started on %s (observing: time-pos, pause, seeking, eof-reached)", el.socketPath)
	return nil
}

// Stop terminates the event listener.
func (el *EventListener) Stop() {
	el.mu.Lock()
	defer el.mu.Unlock()

	if !el.listening {
		return
	}

	close(el.stopCh)
	if el.conn != nil {
		el.conn.Close()
	}
	el.listening = false
}

// readLoop continuously reads events from the persistent mpv connection.
// mpv sends newline-delimited JSON events when observed properties change.
func (el *EventListener) readLoop() {
	defer func() {
		el.mu.Lock()
		el.listening = false
		el.mu.Unlock()
	}()

	buf := make([]byte, 4096)
	var remainder []byte

	for {
		select {
		case <-el.stopCh:
			return
		default:
		}

		// Set read deadline to avoid blocking forever
		if err := el.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return
		}

		n, err := el.conn.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
				continue // timeout is normal, keep listening
			}
			log.Warnf("event listener read error: %v", err)
			return
		}

		// mpv sends multiple JSON objects separated by newlines
		data := append(remainder, buf[:n]...)
		remainder = nil

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Last incomplete line goes to remainder for next read
			if i == len(lines)-1 && !strings.HasSuffix(string(data), "\n") {
				remainder = []byte(line)
				continue
			}

			el.processEvent(line)
		}
	}
}

// processEvent parses and dispatches a single mpv event JSON line.
func (el *EventListener) processEvent(line string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return // Skip unparseable lines
	}

	// Property change events have "event": "property-change" and "name" + "data"
	if eventType, ok := event["event"].(string); ok {
		switch eventType {
		case "property-change":
			name, _ := event["name"].(string)
			data := event["data"]
			if name != "" && el.callback != nil {
				el.callback(name, data)
			}
		default:
			// Forward other events (e.g., "playback-restart", "end-file")
			if el.callback != nil {
				el.callback(eventType, event)
			}
		}
	}
}
