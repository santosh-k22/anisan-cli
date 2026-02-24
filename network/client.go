// Package network provides a pre-configured, optimized HTTP client for concurrent provider communication.
package network

import (
	"net/http"
	"time"
)

// Client is the singleton HTTP client shared across the application for efficient resource utilization.
// It is configured with increased concurrency limits and reasonable timeouts tailored for scraping workflows.
var Client = &http.Client{
	Timeout:   time.Minute,
	Transport: newTransport(),
}

// newTransport initializes a tuned http.Transport with optimized pool and timeout parameters.
func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 100
	t.MaxConnsPerHost = 200
	t.IdleConnTimeout = 30 * time.Second
	t.ResponseHeaderTimeout = 30 * time.Second
	t.ExpectContinueTimeout = 30 * time.Second
	return t
}
