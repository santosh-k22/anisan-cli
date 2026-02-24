// Package player defines a unified abstraction layer for media playback engines.
package player

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// ipcCommand is the JSON structure sent to mpv's IPC socket.
type ipcCommand struct {
	Command []interface{} `json:"command"`
}

// ipcResponse is the JSON structure received from mpv's IPC socket.
type ipcResponse struct {
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

const (
	maxRetries   = 3
	retryDelay   = 100 * time.Millisecond
	readDeadline = 1 * time.Second
	readBufSize  = 4096
)

// sendCommand sends a JSON-IPC command to mpv via Unix domain socket.
// It implements a retry mechanism for transient connection errors and ensures thread safety.
func (m *MPV) sendCommand(command []interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		result, err := doSendCommand(m.socketPath, command)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("ipc command failed after %d attempts: %w", maxRetries, lastErr)
}

// doSendCommand performs a single IPC command attempt.
func doSendCommand(socketPath string, command []interface{}) (interface{}, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	// Marshal the command
	payload, err := json.Marshal(ipcCommand{Command: command})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	// Send command + newline (mpv requires newline-delimited JSON)
	_, err = conn.Write(append(payload, '\n'))
	if err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	// Read response with timeout
	if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	buf := make([]byte, readBufSize)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	// Parse response
	var resp ipcResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if resp.Error != "" && resp.Error != "success" {
		return nil, fmt.Errorf("mpv error: %s", resp.Error)
	}

	return resp.Data, nil
}
