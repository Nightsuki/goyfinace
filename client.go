package yfinance

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
	defaultQuery1URL = "https://query1.finance.yahoo.com"
	defaultQuery2URL = "https://query2.finance.yahoo.com"
	defaultUserAgent = "Mozilla/5.0 (compatible; goyfinace/0.1; +https://github.com/Nightsuki/goyfinace)"
)

// Client is a Yahoo Finance HTTP client.
type Client struct {
	// HTTPClient is used for all requests. A nil value is replaced by a client
	// with a 15 second timeout.
	HTTPClient *http.Client
	// Query1URL and Query2URL allow tests and advanced callers to override the
	// Yahoo host. Leave empty for the default Yahoo endpoints.
	Query1URL string
	Query2URL string
	// UserAgent is sent on every request. Leave empty for the package default.
	UserAgent string
}

// NewClient returns a client configured for Yahoo Finance public endpoints.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		HTTPClient: httpClient,
		Query1URL:  defaultQuery1URL,
		Query2URL:  defaultQuery2URL,
		UserAgent:  defaultUserAgent,
	}
}

// Ticker creates a symbol-scoped helper.
func (c *Client) Ticker(symbol string) *Ticker {
	return &Ticker{Symbol: normalizeSymbol(symbol), client: c}
}

func (c *Client) cloneWithDefaults() *Client {
	if c == nil {
		return NewClient(nil)
	}
	cp := *c
	if cp.HTTPClient == nil {
		cp.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if cp.Query1URL == "" {
		cp.Query1URL = defaultQuery1URL
	}
	if cp.Query2URL == "" {
		cp.Query2URL = defaultQuery2URL
	}
	if cp.UserAgent == "" {
		cp.UserAgent = defaultUserAgent
	}
	return &cp
}

func (c *Client) getJSON(ctx context.Context, baseURL, path string, query url.Values, out any) error {
	c = c.cloneWithDefaults()
	endpoint, err := joinURL(baseURL, path, query)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		io.Copy(io.Discard, resp.Body)
		return ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("yfinance: GET %s returned %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	return dec.Decode(out)
}

func joinURL(baseURL, path string, query url.Values) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	p, err := url.JoinPath(u.Path, path)
	if err != nil {
		return "", err
	}
	u.Path = p
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
