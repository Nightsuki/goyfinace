package yfinance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const (
	LookupAll            = "all"
	LookupEquity         = "equity"
	LookupMutualFund     = "mutualfund"
	LookupETF            = "etf"
	LookupIndex          = "index"
	LookupFuture         = "future"
	LookupCurrency       = "currency"
	LookupCryptocurrency = "cryptocurrency"
)

// Lookup searches Yahoo Finance instruments by type.
func (c *Client) Lookup(ctx context.Context, query string, lookupType string, count int) (map[string]any, error) {
	if query == "" {
		return nil, fmt.Errorf("yfinance: empty lookup query")
	}
	if lookupType == "" {
		lookupType = LookupAll
	}
	if count <= 0 {
		count = 25
	}
	if err := validateMax("lookup count", count, maxLookupCount); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("query", query)
	q.Set("type", lookupType)
	q.Set("start", "0")
	q.Set("count", strconv.Itoa(count))
	q.Set("formatted", "false")
	q.Set("fetchPricingData", "true")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v1/finance/lookup", q, &resp)
	return resp, err
}

// LookupISIN resolves an ISIN or other query through Yahoo search.
func (c *Client) LookupISIN(ctx context.Context, isin string) (*SearchResponse, error) {
	return c.Search(ctx, isin, 1, 0)
}

// ScreenRequest is a Yahoo screener request body.
type ScreenRequest struct {
	Offset     int            `json:"offset,omitempty"`
	Count      int            `json:"count,omitempty"`
	Size       int            `json:"size,omitempty"`
	SortField  string         `json:"sortField,omitempty"`
	SortType   string         `json:"sortType,omitempty"`
	QuoteType  string         `json:"quoteType,omitempty"`
	Query      map[string]any `json:"query,omitempty"`
	UserID     string         `json:"userId,omitempty"`
	UserIDType string         `json:"userIdType,omitempty"`
}

// ScreenerQuery is a composable Yahoo screener query expression.
type ScreenerQuery struct {
	Operator string `json:"operator"`
	Operands []any  `json:"operands"`
}

// Map converts a query expression into the raw map shape accepted by Screen.
func (q ScreenerQuery) Map() map[string]any {
	return map[string]any{"operator": q.Operator, "operands": q.Operands}
}

// Screener builds a raw Yahoo screener query expression.
func Screener(operator string, operands ...any) ScreenerQuery {
	return ScreenerQuery{Operator: operator, Operands: operands}
}

// Eq builds an equality screener filter.
func Eq(field string, value any) ScreenerQuery {
	return Screener("eq", field, value)
}

// GT builds a greater-than screener filter.
func GT(field string, value any) ScreenerQuery {
	return Screener("gt", field, value)
}

// GTE builds a greater-than-or-equal screener filter.
func GTE(field string, value any) ScreenerQuery {
	return Screener("gte", field, value)
}

// LT builds a less-than screener filter.
func LT(field string, value any) ScreenerQuery {
	return Screener("lt", field, value)
}

// LTE builds a less-than-or-equal screener filter.
func LTE(field string, value any) ScreenerQuery {
	return Screener("lte", field, value)
}

// Between builds a between screener filter.
func Between(field string, low any, high any) ScreenerQuery {
	return Screener("btwn", field, low, high)
}

// And combines screener filters with a logical AND.
func And(filters ...ScreenerQuery) ScreenerQuery {
	operands := make([]any, 0, len(filters))
	for _, filter := range filters {
		operands = append(operands, filter.Map())
	}
	return Screener("and", operands...)
}

// Or combines screener filters with a logical OR.
func Or(filters ...ScreenerQuery) ScreenerQuery {
	operands := make([]any, 0, len(filters))
	for _, filter := range filters {
		operands = append(operands, filter.Map())
	}
	return Screener("or", operands...)
}

// EquityQuery builds a Yahoo equity screener query.
func EquityQuery(filters ...ScreenerQuery) map[string]any {
	return queryWithQuoteType("EQUITY", filters...)
}

// FundQuery builds a Yahoo mutual-fund screener query.
func FundQuery(filters ...ScreenerQuery) map[string]any {
	return queryWithQuoteType("MUTUALFUND", filters...)
}

// ETFQuery builds a Yahoo ETF screener query.
func ETFQuery(filters ...ScreenerQuery) map[string]any {
	return queryWithQuoteType("ETF", filters...)
}

func queryWithQuoteType(quoteType string, filters ...ScreenerQuery) map[string]any {
	all := append([]ScreenerQuery{Eq("quoteType", quoteType)}, filters...)
	return And(all...).Map()
}

// Screen runs a custom Yahoo screener query.
func (c *Client) Screen(ctx context.Context, req ScreenRequest) (map[string]any, error) {
	if req.Count == 0 && req.Size == 0 {
		req.Size = 100
	}
	if err := validateMax("screener count", req.Count, maxScreenerCount); err != nil {
		return nil, err
	}
	if err := validateMax("screener size", req.Size, maxScreenerCount); err != nil {
		return nil, err
	}
	if req.SortField == "" {
		req.SortField = "ticker"
	}
	if req.SortType == "" {
		req.SortType = "DESC"
	}
	if req.UserIDType == "" {
		req.UserIDType = "guid"
	}
	q := url.Values{}
	q.Set("corsDomain", "finance.yahoo.com")
	q.Set("formatted", "false")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	var resp map[string]any
	err := c.postJSON(ctx, c.cloneWithDefaults().Query1URL, "/v1/finance/screener", q, req, &resp)
	return resp, err
}

// PredefinedScreen runs one of Yahoo's predefined screener queries.
func (c *Client) PredefinedScreen(ctx context.Context, screenID string, count int) (map[string]any, error) {
	if screenID == "" {
		return nil, fmt.Errorf("yfinance: empty screener id")
	}
	if count <= 0 {
		count = 25
	}
	if err := validateMax("screener count", count, maxScreenerCount); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("scrIds", screenID)
	q.Set("count", strconv.Itoa(count))
	q.Set("formatted", "false")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v1/finance/screener/predefined/saved", q, &resp)
	return resp, err
}

// MarketSummary fetches Yahoo market summary data for a market, e.g. "us".
func (c *Client) MarketSummary(ctx context.Context, market string) (map[string]any, error) {
	if market == "" {
		market = "us"
	}
	q := url.Values{}
	q.Set("fields", "shortName,regularMarketPrice,regularMarketChange,regularMarketChangePercent")
	q.Set("formatted", "false")
	q.Set("lang", "en-US")
	q.Set("market", market)
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v6/finance/quote/marketSummary", q, &resp)
	return resp, err
}

// MarketStatus fetches Yahoo market time/status data for a market, e.g. "us".
func (c *Client) MarketStatus(ctx context.Context, market string) (map[string]any, error) {
	if market == "" {
		market = "us"
	}
	q := url.Values{}
	q.Set("formatted", "true")
	q.Set("key", "finance")
	q.Set("lang", "en-US")
	q.Set("market", market)
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v6/finance/markettime", q, &resp)
	return resp, err
}

// Sector fetches Yahoo sector domain data by key, e.g. "technology".
func (c *Client) Sector(ctx context.Context, key string) (map[string]any, error) {
	return c.domain(ctx, "/v1/finance/sectors/"+url.PathEscape(key))
}

// Industry fetches Yahoo industry domain data by key.
func (c *Client) Industry(ctx context.Context, key string) (map[string]any, error) {
	return c.domain(ctx, "/v1/finance/industries/"+url.PathEscape(key))
}

func (c *Client) domain(ctx context.Context, path string) (map[string]any, error) {
	q := url.Values{}
	q.Set("formatted", "true")
	q.Set("withReturns", "true")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, path, q, &resp)
	return resp, err
}

// CalendarQuery is a raw Yahoo visualization calendar query.
type CalendarQuery struct {
	Size          int            `json:"size,omitempty"`
	Offset        int            `json:"offset,omitempty"`
	Query         map[string]any `json:"query,omitempty"`
	SortField     string         `json:"sortField,omitempty"`
	SortType      string         `json:"sortType,omitempty"`
	EntityIDType  string         `json:"entityIdType,omitempty"`
	IncludeFields []string       `json:"includeFields,omitempty"`
}

// CalendarVisualization runs Yahoo's visualization endpoint used by yfinance calendars.
func (c *Client) CalendarVisualization(ctx context.Context, query CalendarQuery) (map[string]any, error) {
	if query.Size == 0 {
		query.Size = 100
	}
	if err := validateMax("calendar size", query.Size, maxCalendarSize); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("lang", "en-US")
	q.Set("region", "US")
	var resp map[string]any
	err := c.postJSON(ctx, c.cloneWithDefaults().Query1URL, "/v1/finance/visualization", q, query, &resp)
	return resp, err
}

// EarningsDates fetches earnings calendar rows for a ticker using Yahoo's visualization endpoint.
func (c *Client) EarningsDates(ctx context.Context, symbol string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 12
	}
	if err := validateMax("earnings dates limit", limit, maxCalendarSize); err != nil {
		return nil, err
	}
	return c.CalendarVisualization(ctx, CalendarQuery{
		Size:          limit,
		Query:         map[string]any{"operator": "eq", "operands": []any{"ticker", normalizeSymbol(symbol)}},
		SortField:     "startdatetime",
		SortType:      "DESC",
		EntityIDType:  "earnings",
		IncludeFields: []string{"startdatetime", "timeZoneShortName", "epsestimate", "epsactual", "epssurprisepct", "eventtype"},
	})
}

// EarningsDates fetches earnings calendar rows for this ticker.
func (t *Ticker) EarningsDates(ctx context.Context, limit int) (map[string]any, error) {
	return t.c().EarningsDates(ctx, t.Symbol, limit)
}
