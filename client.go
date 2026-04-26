package yfinance

import (
	"bytes"
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
	defaultRootURL   = "https://finance.yahoo.com"
	defaultISINURL   = "https://markets.businessinsider.com"
	defaultStreamURL = "wss://streamer.finance.yahoo.com/?version=2"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	maxTextBodyBytes = 2 << 20
)

// DefaultUserAgents mirrors the browser user-agent set used by yfinance.
var DefaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:135.0) Gecko/20100101 Firefox/135.0",
	"Mozilla/5.0 (X11; Linux i686; rv:135.0) Gecko/20100101 Firefox/135.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/131.0.2903.86",
}

// Client is a Yahoo Finance HTTP client.
type Client struct {
	// HTTPClient is used for all requests. A nil value is replaced by a client
	// with a 15 second timeout.
	HTTPClient *http.Client
	// Query1URL and Query2URL allow tests and advanced callers to override the
	// Yahoo host. Leave empty for the default Yahoo endpoints.
	Query1URL string
	Query2URL string
	// RootURL is used for Yahoo Finance frontend JSON endpoints.
	RootURL string
	// ISINURL is used only by the best-effort ISIN helper. Yahoo does not
	// expose a stable ticker-to-ISIN endpoint, so this is intentionally separate.
	ISINURL string
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
		RootURL:    defaultRootURL,
		ISINURL:    defaultISINURL,
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
	if cp.RootURL == "" {
		cp.RootURL = defaultRootURL
	}
	if cp.ISINURL == "" {
		cp.ISINURL = defaultISINURL
	}
	if cp.UserAgent == "" {
		cp.UserAgent = defaultUserAgent
	}
	return &cp
}

func (c *Client) getJSON(ctx context.Context, baseURL, path string, query url.Values, out any) error {
	return c.doJSON(ctx, http.MethodGet, baseURL, path, query, nil, out)
}

func (c *Client) getText(ctx context.Context, baseURL, path string, query url.Values) (string, error) {
	c = c.cloneWithDefaults()
	endpoint, err := joinURL(baseURL, path, query)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		io.Copy(io.Discard, resp.Body)
		return "", ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("yfinance: GET %s returned %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	limited := io.LimitReader(resp.Body, maxTextBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(body) > maxTextBodyBytes {
		return "", fmt.Errorf("yfinance: GET %s response exceeded %d bytes", path, maxTextBodyBytes)
	}
	return string(body), nil
}

func (c *Client) postJSON(ctx context.Context, baseURL, path string, query url.Values, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, baseURL, path, query, body, out)
}

func (c *Client) doJSON(ctx context.Context, method, baseURL, path string, query url.Values, body any, out any) error {
	c = c.cloneWithDefaults()
	endpoint, err := joinURL(baseURL, path, query)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

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
		return fmt.Errorf("yfinance: %s %s returned %s: %s", method, path, resp.Status, strings.TrimSpace(string(body)))
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
