// Package plaud provides a Go client for the unofficial Plaud API.
//
// Usage:
//
//	client := plaud.New("your-bearer-token")
//	files, err := client.ListFiles(ctx, nil)
//	text, err := client.GetTranscriptText(ctx, "file-id")
package plaud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the default Plaud API base URL.
	DefaultBaseURL = "https://api.plaud.ai"
)

// Client is a Plaud API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// New creates a new Plaud API client.
// The token should be the bearer token value (with or without the "bearer " prefix).
func New(token string, opts ...Option) *Client {
	if !strings.HasPrefix(token, "bearer ") {
		token = "bearer " + token
	}
	c := &Client{
		baseURL: DefaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do executes an HTTP request with authorization headers and returns the raw response.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	u := c.baseURL + path
	if query != nil {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("plaud: creating request: %w", err)
	}

	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("app-platform", "web")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plaud: executing request: %w", err)
	}

	return resp, nil
}

// doJSON executes an HTTP request and decodes the JSON response into v.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body io.Reader, v interface{}) error {
	resp, err := c.Do(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plaud: API error %d: %s", resp.StatusCode, string(b))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("plaud: decoding response: %w", err)
		}
	}
	return nil
}
