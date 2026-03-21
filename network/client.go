package network

import (
	"net/http"
	"time"
)

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
