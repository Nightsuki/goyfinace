package yfinance

import "context"

// Tickers is a multi-symbol helper similar to yfinance.Tickers.
type Tickers struct {
	Symbols []string
	client  *Client
}

// Tickers creates a multi-symbol helper.
func (c *Client) Tickers(symbols ...string) *Tickers {
	return &Tickers{Symbols: normalizeSymbols(symbols), client: c.cloneWithDefaults()}
}

// Ticker returns a symbol-scoped helper from this collection.
func (t *Tickers) Ticker(symbol string) *Ticker {
	return t.client.Ticker(symbol)
}

// Download downloads history for all symbols in this collection.
func (t *Tickers) Download(ctx context.Context, params HistoryParams) []DownloadResult {
	if t == nil {
		return nil
	}
	return t.client.Download(ctx, t.Symbols, params)
}

// Quotes fetches quote rows for all symbols in this collection.
func (t *Tickers) Quotes(ctx context.Context) ([]Quote, error) {
	if t == nil {
		return nil, ErrNoResult
	}
	return t.client.Quote(ctx, t.Symbols...)
}

// Market is a market-scoped helper for summary and status data.
type Market struct {
	Name   string
	client *Client
}

// Market creates a market-scoped helper, for example "us".
func (c *Client) Market(name string) *Market {
	if name == "" {
		name = "us"
	}
	return &Market{Name: name, client: c.cloneWithDefaults()}
}

// Summary fetches market summary data.
func (m *Market) Summary(ctx context.Context) (map[string]any, error) {
	return m.client.MarketSummary(ctx, m.Name)
}

// Status fetches market status/time data.
func (m *Market) Status(ctx context.Context) (map[string]any, error) {
	return m.client.MarketStatus(ctx, m.Name)
}

// Calendars is a helper for Yahoo calendar endpoints.
type Calendars struct {
	client *Client
}

// Calendars creates a calendar helper.
func (c *Client) Calendars() *Calendars {
	return &Calendars{client: c.cloneWithDefaults()}
}

// Visualization runs a raw Yahoo calendar visualization query.
func (cals *Calendars) Visualization(ctx context.Context, query CalendarQuery) (map[string]any, error) {
	return cals.client.CalendarVisualization(ctx, query)
}

// Earnings fetches earnings calendar rows. When symbols are provided, the
// first symbol is used for the ticker-specific endpoint.
func (cals *Calendars) Earnings(ctx context.Context, limit int, symbols ...string) (map[string]any, error) {
	if len(symbols) > 0 && normalizeSymbol(symbols[0]) != "" {
		return cals.client.EarningsDates(ctx, symbols[0], limit)
	}
	if limit <= 0 {
		limit = 100
	}
	if err := validateMax("earnings calendar limit", limit, maxCalendarSize); err != nil {
		return nil, err
	}
	return cals.client.CalendarVisualization(ctx, CalendarQuery{
		Size:         limit,
		SortField:    "startdatetime",
		SortType:     "ASC",
		EntityIDType: "earnings",
	})
}
