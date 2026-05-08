package yfinance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultQuery1URL       = "https://query1.finance.yahoo.com"
	defaultQuery2URL       = "https://query2.finance.yahoo.com"
	defaultRootURL         = "https://finance.yahoo.com"
	defaultISINURL         = "https://markets.businessinsider.com"
	defaultStreamURL       = "wss://streamer.finance.yahoo.com/?version=2"
	defaultCookiePrimeURL  = "https://fc.yahoo.com"
	defaultUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	maxTextBodyBytes       = 2 << 20
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
	// CookiePrimeURL is the small Yahoo endpoint visited only to set the
	// A1/A3 session cookies before fetching a crumb. yfinance's Python
	// implementation uses https://fc.yahoo.com (a 404-returning endpoint
	// that still sets cookies); the full https://finance.yahoo.com/ home
	// page is multiple megabytes and is the wrong choice for priming.
	CookiePrimeURL string
	// UserAgent is sent on every request. Leave empty for the package default.
	UserAgent string
	// Crumb is the bot-detection token Yahoo requires on quoteSummary and
	// related endpoints. Use Authenticate to populate it automatically; it is
	// then auto-appended to outgoing requests.
	Crumb string

	// Logger receives debug-level events for each request and response when
	// non-nil. Mirrors yfinance's enable_debug_mode() but uses slog so callers
	// can plug in any handler.
	Logger *slog.Logger

	// Retries bounds how many times a transient failure (5xx, ErrRateLimited,
	// network error) is retried with exponential backoff. Zero means a single
	// attempt with no retries. Mirrors yfinance's session-level retry behavior.
	Retries int

	// RetryBackoff is the base delay between retries; each retry doubles the
	// previous wait. Zero means 250ms.
	RetryBackoff time.Duration

	// Limiter, when non-nil, gates every outgoing request through Wait(ctx).
	// Use NewRateLimiter for a simple token-bucket implementation, or supply
	// any Limiter implementation (e.g. golang.org/x/time/rate.Limiter).
	Limiter Limiter

	// Cache, when non-nil, caches GET-JSON responses keyed by the full URL.
	// Use NewMemoryCache for a simple in-process implementation. CacheTTL
	// controls per-entry expiry; a zero value means cached entries never
	// expire.
	Cache    Cache
	CacheTTL time.Duration

	auth *authState
}

// Limiter is the minimal interface a request-rate limiter must implement.
// It is satisfied by *RateLimiter in this package and by
// golang.org/x/time/rate.Limiter.
type Limiter interface {
	Wait(ctx context.Context) error
}

// authState holds shared authentication mutexes/cache so cloned Client values
// reuse the same lock and crumb.
type authState struct {
	mu    sync.Mutex
	crumb string
}

// NewClient returns a client configured for Yahoo Finance public endpoints.
//
// If httpClient is nil, a default client is created with a 15-second timeout
// and an in-memory cookie jar. The cookie jar is required for Yahoo's
// crumb-based authentication; supply your own jar by setting
// HTTPClient.Jar yourself when passing a custom http.Client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		jar, _ := cookiejar.New(nil)
		httpClient = &http.Client{Timeout: 15 * time.Second, Jar: jar}
	} else if httpClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		httpClient.Jar = jar
	}
	return &Client{
		HTTPClient:     httpClient,
		Query1URL:      defaultQuery1URL,
		Query2URL:      defaultQuery2URL,
		RootURL:        defaultRootURL,
		ISINURL:        defaultISINURL,
		CookiePrimeURL: defaultCookiePrimeURL,
		UserAgent:      defaultUserAgent,
		auth:           &authState{},
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
	cp := Client{
		HTTPClient:     c.HTTPClient,
		Query1URL:      c.Query1URL,
		Query2URL:      c.Query2URL,
		RootURL:        c.RootURL,
		ISINURL:        c.ISINURL,
		CookiePrimeURL: c.CookiePrimeURL,
		UserAgent:      c.UserAgent,
		Crumb:          c.Crumb,
		Logger:         c.Logger,
		Retries:        c.Retries,
		RetryBackoff:   c.RetryBackoff,
		Limiter:        c.Limiter,
		Cache:          c.Cache,
		CacheTTL:       c.CacheTTL,
		auth:           c.auth,
	}
	if cp.HTTPClient == nil {
		jar, _ := cookiejar.New(nil)
		cp.HTTPClient = &http.Client{Timeout: 15 * time.Second, Jar: jar}
	} else if cp.HTTPClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		cp.HTTPClient.Jar = jar
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
	if cp.CookiePrimeURL == "" {
		cp.CookiePrimeURL = defaultCookiePrimeURL
	}
	if cp.UserAgent == "" {
		cp.UserAgent = defaultUserAgent
	}
	if cp.auth == nil {
		cp.auth = &authState{}
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
	var out string
	err = c.withRetry(ctx, http.MethodGet, path, func() error {
		if err := c.gate(ctx); err != nil {
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
		if resp.StatusCode == http.StatusTooManyRequests {
			io.Copy(io.Discard, resp.Body)
			return ErrRateLimited
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("yfinance: GET %s returned %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
		}
		limited := io.LimitReader(resp.Body, maxTextBodyBytes+1)
		body, readErr := io.ReadAll(limited)
		if readErr != nil {
			return readErr
		}
		if len(body) > maxTextBodyBytes {
			return fmt.Errorf("yfinance: GET %s response exceeded %d bytes", path, maxTextBodyBytes)
		}
		out = string(body)
		return nil
	})
	return out, err
}

func (c *Client) postJSON(ctx context.Context, baseURL, path string, query url.Values, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, baseURL, path, query, body, out)
}

func (c *Client) doJSON(ctx context.Context, method, baseURL, path string, query url.Values, body any, out any) error {
	c = c.cloneWithDefaults()
	cacheable := method == http.MethodGet && c.Cache != nil
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	return c.withRetry(ctx, method, path, func() error {
		// Re-evaluate crumb each attempt so a refreshed crumb after 401 is
		// applied to the retry. The endpoint is rebuilt accordingly.
		q := query
		if requestNeedsCrumb(baseURL, path) {
			if err := c.EnsureCrumb(ctx); err != nil {
				return err
			}
			q = appendCrumb(q, c.Crumb)
		}
		endpoint, err := joinURL(baseURL, path, q)
		if err != nil {
			return err
		}
		if cacheable {
			if cached, ok := c.Cache.Get(endpoint); ok {
				if c.Logger != nil {
					c.Logger.Debug("yfinance cache hit",
						slog.String("method", method),
						slog.String("url", endpoint))
				}
				dec := json.NewDecoder(bytes.NewReader(cached))
				dec.UseNumber()
				return dec.Decode(out)
			}
		}
		if err := c.gate(ctx); err != nil {
			return err
		}
		var reader io.Reader
		if data != nil {
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
		c.logRequest(req)
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		c.logResponse(req, resp)
		if resp.StatusCode == http.StatusTooManyRequests {
			io.Copy(io.Discard, resp.Body)
			return ErrRateLimited
		}
		// A 401 on a crumb-protected path means the cached crumb expired;
		// drop it so the next attempt re-authenticates from scratch.
		if resp.StatusCode == http.StatusUnauthorized && requestNeedsCrumb(baseURL, path) {
			io.Copy(io.Discard, resp.Body)
			c.invalidateCrumb()
			return errCrumbExpired
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			// Yahoo returns 404 with "No fundamentals data found" when an
			// otherwise-valid request lands on a symbol/module combo that
			// has no data (e.g. ESG scores for many large caps). yfinance
			// surfaces this as "no data" rather than an error; mirror that
			// by returning ErrNoResult.
			if resp.StatusCode == http.StatusNotFound &&
				strings.Contains(string(payload), "No fundamentals data found") {
				return ErrNoResult
			}
			return &httpError{
				method:  method,
				path:    path,
				status:  resp.StatusCode,
				message: strings.TrimSpace(string(payload)),
			}
		}
		if cacheable {
			payload, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return readErr
			}
			c.Cache.Set(endpoint, payload, c.CacheTTL)
			dec := json.NewDecoder(bytes.NewReader(payload))
			dec.UseNumber()
			return dec.Decode(out)
		}
		dec := json.NewDecoder(resp.Body)
		dec.UseNumber()
		return dec.Decode(out)
	})
}

// httpError carries the HTTP status alongside the message so withRetry can
// classify retryable 5xx vs non-retryable 4xx responses.
type httpError struct {
	method  string
	path    string
	status  int
	message string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("yfinance: %s %s returned %d: %s", e.method, e.path, e.status, e.message)
}

func (c *Client) gate(ctx context.Context) error {
	if c.Limiter == nil {
		return nil
	}
	return c.Limiter.Wait(ctx)
}

func (c *Client) logRequest(req *http.Request) {
	if c.Logger == nil {
		return
	}
	c.Logger.Debug("yfinance request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()))
}

func (c *Client) logResponse(req *http.Request, resp *http.Response) {
	if c.Logger == nil {
		return
	}
	c.Logger.Debug("yfinance response",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Int("status", resp.StatusCode))
}

func (c *Client) withRetry(ctx context.Context, method, path string, attempt func() error) error {
	maxAttempts := c.Retries + 1
	backoff := c.RetryBackoff
	if backoff <= 0 {
		backoff = 250 * time.Millisecond
	}
	var err error
	crumbRefreshed := false
	for i := 0; i < maxAttempts; i++ {
		err = attempt()
		if err == nil {
			return nil
		}
		// Crumb expiry gets one free re-auth retry that does not consume a
		// Retries slot — it's an authentication refresh, not a transient
		// failure.
		if errors.Is(err, errCrumbExpired) && !crumbRefreshed {
			crumbRefreshed = true
			i--
			continue
		}
		if !isRetryable(err) || i == maxAttempts-1 {
			return err
		}
		wait := backoff << i
		if c.Logger != nil {
			c.Logger.Debug("yfinance retry",
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("attempt", i+1),
				slog.Duration("wait", wait),
				slog.String("err", err.Error()))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

// errCrumbExpired signals that a crumb-protected endpoint returned 401 and
// the cached crumb has been invalidated. withRetry treats this as retryable
// so the next attempt re-runs Authenticate transparently.
var errCrumbExpired = errors.New("yfinance: crumb expired, re-authenticated")

func (c *Client) invalidateCrumb() {
	c.Crumb = ""
	if c.auth != nil {
		c.auth.mu.Lock()
		c.auth.crumb = ""
		c.auth.mu.Unlock()
	}
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var he *httpError
	if errors.As(err, &he) && he.status >= 500 && he.status < 600 {
		return true
	}
	// Network errors (DNS, connection reset) are reported as plain errors from
	// http.Client; retry conservatively if the context is still alive.
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Temporary() {
			return true
		}
	}
	return false
}

func joinURL(baseURL, path string, query url.Values) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	// Use the *URL method* JoinPath (not the package func url.JoinPath) so
	// the resolved path correctly populates both u.Path (decoded) and
	// u.RawPath (encoded). The package func returns an already-encoded
	// string that, when assigned back to u.Path, gets double-encoded by
	// u.String() — turning "%5E" (^) into "%255E" and breaking any
	// symbol that contains a percent-escapable character (e.g. ^VIX,
	// ^TNX, ^GSPC indices, BRK.B class shares, foreign tickers with
	// non-ASCII letters).
	u = u.JoinPath(path)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
