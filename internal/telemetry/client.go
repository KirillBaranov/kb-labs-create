// Package telemetry provides anonymous, fire-and-forget usage analytics.
//
// The Client is safe for concurrent use. All Track() calls dispatch an HTTP
// POST in a background goroutine and never block the caller. Network errors
// are silently discarded — telemetry must never interfere with the install.
//
// Use Nop() to obtain a no-op client when consent is not given. Callers can
// call Track/Flush unconditionally without checking consent themselves.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Endpoint is the URL where events are POSTed.
// TODO: replace with real backend (Cloudflare Worker, etc.)
var Endpoint = "https://telemetry.kb-labs.dev/v1/events"

// Event is the JSON body sent to the telemetry endpoint.
type Event struct {
	DeviceID string            `json:"deviceId"`
	Name     string            `json:"event"`
	Version  string            `json:"version"`
	OS       string            `json:"os"`
	Arch     string            `json:"arch"`
	TS       string            `json:"ts"`
	Props    map[string]string `json:"props,omitempty"`
}

// Client sends anonymous telemetry events over HTTP.
type Client struct {
	endpoint string
	deviceID string
	version  string

	mu    sync.Mutex
	props map[string]string // shared properties attached to every event
	wg    sync.WaitGroup
	nop   bool
}

// New creates a live telemetry client that posts to endpoint.
func New(endpoint, deviceID, version string) *Client {
	return &Client{
		endpoint: endpoint,
		deviceID: deviceID,
		version:  version,
		props:    make(map[string]string),
	}
}

// Nop returns a client that silently discards all events.
func Nop() *Client {
	return &Client{nop: true}
}

// Set attaches a shared property to all future events (e.g. "pm", "services").
func (c *Client) Set(key, value string) {
	if c.nop {
		return
	}
	c.mu.Lock()
	c.props[key] = value
	c.mu.Unlock()
}

// Track sends an event in a background goroutine.
// extra properties are merged on top of shared properties set via Set().
func (c *Client) Track(event string, extra map[string]string) {
	if c.nop {
		return
	}

	// Snapshot shared props under lock.
	c.mu.Lock()
	merged := make(map[string]string, len(c.props)+len(extra))
	for k, v := range c.props {
		merged[k] = v
	}
	c.mu.Unlock()
	for k, v := range extra {
		merged[k] = v
	}

	e := Event{
		DeviceID: c.deviceID,
		Name:     event,
		Version:  c.version,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		TS:       time.Now().UTC().Format(time.RFC3339),
		Props:    merged,
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.send(e)
	}()
}

// Flush blocks until all pending Track() goroutines have finished.
// Call this before process exit to give in-flight requests a chance to land.
func (c *Client) Flush() {
	if c.nop {
		return
	}
	c.wg.Wait()
}

func (c *Client) send(e Event) {
	body, err := json.Marshal(e)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
