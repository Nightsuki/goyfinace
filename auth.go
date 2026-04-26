package yfinance

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// crumbProtectedPaths lists URL path substrings whose endpoints require a
// Yahoo crumb when the client is operating against the real Yahoo hosts.
// Tests using httptest with overridden Query1URL/Query2URL skip the auto-
// authentication step because the test server has no crumb endpoint.
var crumbProtectedPaths = []string{
	"/v10/finance/quoteSummary/",
	"/v7/finance/quote",
	"/v7/finance/options/",
	"/v6/finance/quote",
	"/ws/fundamentals-timeseries/",
	"/v1/finance/sectors/",
	"/v1/finance/industries/",
}

// Authenticate primes the client's cookie jar by visiting Yahoo Finance and
// then fetches a fresh crumb token. After a successful call, c.Crumb is set
// and is automatically appended to subsequent requests that target Yahoo's
// crumb-protected endpoints.
//
// Authenticate is safe to call multiple times concurrently; only one fetch
// runs at a time and others observe the cached value.
func (c *Client) Authenticate(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.auth == nil {
		c.auth = &authState{}
	}
	c.auth.mu.Lock()
	defer c.auth.mu.Unlock()
	if c.auth.crumb != "" {
		c.Crumb = c.auth.crumb
		return nil
	}
	// Use a defaulted view for URL/UA/HTTPClient values, but write the result
	// back to the receiver so callers can observe c.Crumb.
	defaulted := c.cloneWithDefaults()
	if err := defaulted.primeCookies(ctx, defaulted.CookiePrimeURL, "/"); err != nil {
		return err
	}
	crumb, err := defaulted.getText(ctx, defaulted.Query2URL, "/v1/test/getcrumb", nil)
	if err != nil {
		return err
	}
	crumb = strings.TrimSpace(crumb)
	if crumb == "" {
		return ErrNoResult
	}
	c.auth.crumb = crumb
	c.Crumb = crumb
	return nil
}

// EnsureCrumb calls Authenticate when the client does not yet have a crumb.
// It is invoked automatically before crumb-protected requests when the client
// targets the real Yahoo hosts.
func (c *Client) EnsureCrumb(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.Crumb != "" {
		return nil
	}
	if c.auth == nil {
		c.auth = &authState{}
	}
	c.auth.mu.Lock()
	cached := c.auth.crumb
	c.auth.mu.Unlock()
	if cached != "" {
		c.Crumb = cached
		return nil
	}
	return c.Authenticate(ctx)
}

func pathRequiresCrumb(path string) bool {
	for _, p := range crumbProtectedPaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

func appendCrumb(query url.Values, crumb string) url.Values {
	if crumb == "" || query.Get("crumb") != "" {
		return query
	}
	out := url.Values{}
	for k, v := range query {
		out[k] = v
	}
	out.Set("crumb", crumb)
	return out
}

// hostMatchesYahoo reports whether base URL points at the real Yahoo hosts.
// Tests using httptest override the URLs and should skip auto-auth.
func hostMatchesYahoo(baseURL string) bool {
	return strings.Contains(baseURL, "yahoo.com")
}

// requestNeedsCrumb is exposed for tests; it intentionally checks both the
// host and the path.
func requestNeedsCrumb(baseURL, path string) bool {
	return hostMatchesYahoo(baseURL) && pathRequiresCrumb(path)
}

// primeCookies issues a GET against baseURL+path, lets the configured
// cookie jar capture any Set-Cookie headers, then drains and discards the
// response body without enforcing a size cap. Cookie priming does not need
// the body content — the jar is populated from response headers — so the
// 2 MiB cap that getText enforces would otherwise reject Yahoo's multi-MB
// home page and break crumb-based authentication.
func (c *Client) primeCookies(ctx context.Context, baseURL, path string) error {
	endpoint, err := joinURL(baseURL, path, nil)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.UserAgent)
	c.logRequest(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.logResponse(req, resp)
	// Drain the body so the connection can be reused, but don't buffer it.
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	// fc.yahoo.com replies with 404 yet still sets cookies on the jar; only
	// 5xx and unrecognized statuses count as priming failures.
	if resp.StatusCode >= 500 {
		return fmt.Errorf("yfinance: cookie prime %s returned %s", path, resp.Status)
	}
	return nil
}
