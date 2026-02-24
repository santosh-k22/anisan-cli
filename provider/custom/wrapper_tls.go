// Package custom provides the TLS-spoofed HTTP client for Lua scripts.
//
// registerTLSClient injects a custom HTTP client with uTLS fingerprinting enabled.
// When the Lua script calls http.get(), the underlying Go engine
// executes the request using the spoofed TLS fingerprint."
//
// This wrapper leverages refraction-networking/utls to implement TLS fingerprint
// emulation, specifically mimicking Chrome's Client Hello signature. This is
// a critical requirement for bypassing sophisticated anti-bot challenges
// (e.g., Cloudflare, DDoS-Guard) that reject standard Go HTTP clients.
//
// Fingerprint Selection:
// uTLS HelloChrome_120 is used as it provides a modern, stable fingerprint
// that matches prevalent browser traffic.
//
// Protocol Negotiation (ALPN):
// The implementation performs automatic protocol detection. It first attempts
// an HTTP/2 connection (preferred by modern CDNs). If the handshake fails or
// the server only supports HTTP/1.1, it transparently falls back to a
// standard H1 transport with forced protocol advertisement.
//
// Lua API:
//
//	http_tls.get(url)              → returns body string
//	http_tls.get(url, headers_tbl) → returns body string with custom headers
//	http_tls.request(options_tbl)  → returns {status, body, headers}
package custom

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anisan-cli/anisan/internal/cache"
	utls "github.com/refraction-networking/utls"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/net/http2"
)

const httpTimeout = 30 * time.Second

// registerTLSClient injects the "http_tls" global module into the Lua state.
// This is called during source loading in loader.go.
func registerTLSClient(L *lua.LState) {
	mod := L.NewTable()

	// http_tls.get(url [, headers_table]) → body_string
	L.SetField(mod, "get", L.NewFunction(httpTLSGet))

	// http_tls.request({method, url, headers, body}) → {status, body, headers}
	L.SetField(mod, "request", L.NewFunction(httpTLSRequest))

	L.SetGlobal("http_tls", mod)
}

// httpTLSGet implements http_tls.get(url [, headers]) → body string
func httpTLSGet(L *lua.LState) int {
	url := L.CheckString(1)
	headersTable := L.OptTable(2, nil)

	headers := make(map[string]string)
	if headersTable != nil {
		headersTable.ForEach(func(k, v lua.LValue) {
			headers[k.String()] = v.String()
		})
	}

	body, _, err := doTLSRequest("GET", url, headers, "")
	if err != nil {
		L.RaiseError("http_tls.get failed: %s", err.Error())
		return 0
	}

	L.Push(lua.LString(body))
	return 1
}

// httpTLSRequest implements http_tls.request(options) → {status, body, headers}
func httpTLSRequest(L *lua.LState) int {
	opts := L.CheckTable(1)

	method := getStringField(opts, "method", "GET")
	url := getStringField(opts, "url", "")
	reqBody := getStringField(opts, "body", "")

	if url == "" {
		L.RaiseError("http_tls.request: url is required")
		return 0
	}

	shouldCache := false
	if cacheVal := opts.RawGetString("cache"); cacheVal != lua.LNil {
		shouldCache = lua.LVAsBool(cacheVal)
	}

	headers := make(map[string]string)
	headersTbl := opts.RawGetString("headers")
	if tbl, ok := headersTbl.(*lua.LTable); ok {
		tbl.ForEach(func(k, v lua.LValue) {
			headers[k.String()] = v.String()
		})
	}

	type tlsCacheEntry struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}

	var cacheKey string
	if shouldCache {
		cacheKey = cache.GenerateKey(url+reqBody, method)
		var entry tlsCacheEntry
		if cache.Read(cacheKey, &entry) {
			result := L.NewTable()
			L.SetField(result, "status", lua.LNumber(entry.Status))
			L.SetField(result, "body", lua.LString(entry.Body))
			L.Push(result)
			return 1
		}
	}

	respBody, statusCode, err := doTLSRequest(method, url, headers, reqBody)
	if err != nil {
		L.RaiseError("http_tls.request failed: %s", err.Error())
		return 0
	}

	if shouldCache && statusCode == 200 {
		entry := tlsCacheEntry{
			Status: statusCode,
			Body:   respBody,
		}
		_ = cache.Write(cacheKey, entry)
	}

	result := L.NewTable()
	L.SetField(result, "status", lua.LNumber(statusCode))
	L.SetField(result, "body", lua.LString(respBody))
	L.Push(result)
	return 1
}

// getStringField is a helper to get a string field from a Lua table with a default.
func getStringField(tbl *lua.LTable, key string, def string) string {
	val := tbl.RawGetString(key)
	if val == lua.LNil {
		return def
	}
	return val.String()
}

// h2Transport is a shared HTTP/2 transport for servers that negotiate h2.
var (
	h2Transport     *http2.Transport
	h2TransportOnce sync.Once
)

func getH2Transport() *http2.Transport {
	h2TransportOnce.Do(func() {
		h2Transport = &http2.Transport{
			// Use custom DialTLSContext to provide utls connections
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialTLS(ctx, network, addr)
			},
		}
	})
	return h2Transport
}

// h1Transport is a shared HTTP/1.1 transport for servers that negotiate http/1.1.
var h1Transport = &http.Transport{
	DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialTLSH1(ctx, network, addr)
	},
}

// doTLSRequest performs an HTTP request with Chrome TLS fingerprint spoofing.
// It automatically handles both H2 and HTTP/1.1 by pre-connecting to determine
// the negotiated protocol, then routing to the appropriate transport.
// Returns (body, statusCode, error).
func doTLSRequest(method, rawURL string, headers map[string]string, body string) (string, int, error) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, rawURL, reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}

	// Set default headers to look like a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// Apply custom headers (overrides defaults)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Try H2 transport first (works for allanime.day and other modern servers)
	client := &http.Client{
		Timeout:   httpTimeout,
		Transport: getH2Transport(),
	}

	resp, err := client.Do(req)
	if err != nil {
		// If H2 fails, fallback to H1 transport
		if body != "" {
			reqBody = strings.NewReader(body) // reset reader
		}
		req2, _ := http.NewRequest(method, rawURL, reqBody)
		req2.Header = req.Header

		h1Client := &http.Client{
			Timeout:   httpTimeout,
			Transport: h1Transport,
		}
		resp, err = h1Client.Do(req2)
		if err != nil {
			return "", 0, fmt.Errorf("request failed: %w", err)
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	return string(respBody), resp.StatusCode, nil
}

// dialTLS creates a TLS connection mimicking Chrome 120's fingerprint.
// Advertises both h2 and http/1.1 (natural Chrome behavior).
func dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	dialer := &net.Dialer{Timeout: httpTimeout}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}, utls.HelloChrome_120)

	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("tls handshake: %w", err)
	}

	return tlsConn, nil
}

// dialTLSH1 creates a TLS connection forcing HTTP/1.1 only (for fallback).
func dialTLSH1(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	dialer := &net.Dialer{Timeout: httpTimeout}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"http/1.1"},
	}, utls.HelloChrome_120)

	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("tls handshake: %w", err)
	}

	return tlsConn, nil
}
