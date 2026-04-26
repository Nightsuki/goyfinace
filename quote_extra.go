package yfinance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Quote is one row returned by Yahoo's v7 quote endpoint.
type Quote struct {
	Symbol                     string
	ShortName                  string
	LongName                   string
	QuoteType                  string
	Exchange                   string
	Currency                   string
	Market                     string
	MarketState                string
	RegularMarketPrice         float64
	RegularMarketChange        float64
	RegularMarketChangePercent float64
	RegularMarketVolume        int64
	Raw                        map[string]any
}

// Quote fetches one or more quote rows from Yahoo's v7 quote endpoint.
func (c *Client) Quote(ctx context.Context, symbols ...string) ([]Quote, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("yfinance: no symbols")
	}
	normalized := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if s := normalizeSymbol(symbol); s != "" {
			normalized = append(normalized, s)
		}
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("yfinance: no symbols")
	}
	if err := validateMax("quote symbols", len(normalized), maxQuoteSymbols); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("symbols", strings.Join(normalized, ","))
	q.Set("formatted", "false")
	var resp struct {
		QuoteResponse struct {
			Result []map[string]any `json:"result"`
			Error  *YahooError      `json:"error"`
		} `json:"quoteResponse"`
	}
	if err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v7/finance/quote", q, &resp); err != nil {
		return nil, err
	}
	if resp.QuoteResponse.Error != nil {
		return nil, resp.QuoteResponse.Error
	}
	out := make([]Quote, 0, len(resp.QuoteResponse.Result))
	for _, row := range resp.QuoteResponse.Result {
		out = append(out, decodeQuote(row))
	}
	return out, nil
}

// Quote fetches this ticker's quote row.
func (t *Ticker) Quote(ctx context.Context) (*Quote, error) {
	quotes, err := t.c().Quote(ctx, t.Symbol)
	if err != nil {
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, ErrNoResult
	}
	return &quotes[0], nil
}

func decodeQuote(row map[string]any) Quote {
	return Quote{
		Symbol:                     stringValue(row["symbol"]),
		ShortName:                  stringValue(row["shortName"]),
		LongName:                   stringValue(row["longName"]),
		QuoteType:                  stringValue(row["quoteType"]),
		Exchange:                   stringValue(row["exchange"]),
		Currency:                   stringValue(row["currency"]),
		Market:                     stringValue(row["market"]),
		MarketState:                stringValue(row["marketState"]),
		RegularMarketPrice:         numberValue(row["regularMarketPrice"]),
		RegularMarketChange:        numberValue(row["regularMarketChange"]),
		RegularMarketChangePercent: numberValue(row["regularMarketChangePercent"]),
		RegularMarketVolume:        int64(numberValue(row["regularMarketVolume"])),
		Raw:                        row,
	}
}

// Calendar fetches Yahoo calendar events for a symbol.
func (c *Client) Calendar(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "calendarEvents")
}

// Calendar fetches Yahoo calendar events for this ticker.
func (t *Ticker) Calendar(ctx context.Context) (map[string]any, error) {
	return t.c().Calendar(ctx, t.Symbol)
}

// SECFilings fetches SEC filing metadata for a symbol.
func (c *Client) SECFilings(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "secFilings")
}

// SECFilings fetches SEC filing metadata for this ticker.
func (t *Ticker) SECFilings(ctx context.Context) (map[string]any, error) {
	return t.c().SECFilings(ctx, t.Symbol)
}

// Sustainability fetches ESG score data for a symbol.
func (c *Client) Sustainability(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "esgScores")
}

// Sustainability fetches ESG score data for this ticker.
func (t *Ticker) Sustainability(ctx context.Context) (map[string]any, error) {
	return t.c().Sustainability(ctx, t.Symbol)
}

// Valuation fetches valuation-related quote summary modules.
func (c *Client) Valuation(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "defaultKeyStatistics", "financialData", "summaryDetail")
}

// Valuation fetches valuation-related quote summary modules for this ticker.
func (t *Ticker) Valuation(ctx context.Context) (map[string]any, error) {
	return t.c().Valuation(ctx, t.Symbol)
}

// UpgradesDowngrades fetches analyst upgrade/downgrade history.
func (c *Client) UpgradesDowngrades(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "upgradeDowngradeHistory")
}

// UpgradesDowngrades fetches analyst upgrade/downgrade history for this ticker.
func (t *Ticker) UpgradesDowngrades(ctx context.Context) (map[string]any, error) {
	return t.c().UpgradesDowngrades(ctx, t.Symbol)
}

// Analysis fetches yfinance-style analyst modules.
func (c *Client) Analysis(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "financialData", "earningsHistory", "earningsTrend", "industryTrend", "sectorTrend", "indexTrend")
}

// Analysis fetches yfinance-style analyst modules for this ticker.
func (t *Ticker) Analysis(ctx context.Context) (map[string]any, error) {
	return t.c().Analysis(ctx, t.Symbol)
}

// AnalystPriceTargets returns the raw financialData module containing target prices.
func (c *Client) AnalystPriceTargets(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "financialData")
}

// AnalystPriceTargets returns the raw financialData module for this ticker.
func (t *Ticker) AnalystPriceTargets(ctx context.Context) (map[string]any, error) {
	return t.c().AnalystPriceTargets(ctx, t.Symbol)
}

// FundProfile fetches fund-specific quote summary modules.
func (c *Client) FundProfile(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "quoteType", "summaryProfile", "fundProfile", "topHoldings")
}

// FundProfile fetches fund-specific quote summary modules for this ticker.
func (t *Ticker) FundProfile(ctx context.Context) (map[string]any, error) {
	return t.c().FundProfile(ctx, t.Symbol)
}

// News fetches ticker news from Yahoo Finance's frontend JSON endpoint.
func (c *Client) News(ctx context.Context, symbol string, count int, tab string) ([]map[string]any, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("yfinance: empty symbol")
	}
	if count <= 0 {
		count = 10
	}
	if err := validateMax("news count", count, maxNewsCount); err != nil {
		return nil, err
	}
	queryRef := map[string]string{
		"all":            "newsAll",
		"news":           "latestNews",
		"press releases": "pressRelease",
	}[strings.ToLower(tab)]
	if queryRef == "" {
		queryRef = "latestNews"
	}
	q := url.Values{}
	q.Set("queryRef", queryRef)
	q.Set("serviceKey", "ncp_fin")
	body := map[string]any{
		"serviceConfig": map[string]any{
			"snippetCount": count,
			"s":            []string{symbol},
		},
	}
	var resp map[string]any
	if err := c.postJSON(ctx, c.cloneWithDefaults().RootURL, "/xhr/ncp", q, body, &resp); err != nil {
		return nil, err
	}
	stream := nestedSlice(resp, "data", "tickerStream", "stream")
	out := make([]map[string]any, 0, len(stream))
	for _, item := range stream {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, isAd := obj["ad"]; isAd {
			continue
		}
		out = append(out, obj)
	}
	return out, nil
}

// News fetches ticker news for this ticker.
func (t *Ticker) News(ctx context.Context, count int, tab string) ([]map[string]any, error) {
	return t.c().News(ctx, t.Symbol, count, tab)
}

// FundamentalsTimeseries fetches raw Yahoo fundamentals timeseries types.
func (c *Client) FundamentalsTimeseries(ctx context.Context, symbol string, types []string, start, end time.Time) (map[string]any, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("yfinance: empty symbol")
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("yfinance: no timeseries types")
	}
	if err := validateMax("timeseries types", len(types), maxTimeseriesTypes); err != nil {
		return nil, err
	}
	if start.IsZero() {
		start = time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC)
	}
	if end.IsZero() {
		end = time.Now().UTC().Add(24 * time.Hour)
	}
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("type", strings.Join(types, ","))
	q.Set("period1", strconv.FormatInt(start.Unix(), 10))
	q.Set("period2", strconv.FormatInt(end.Unix(), 10))
	var resp map[string]any
	err := c.getJSON(ctx, c.cloneWithDefaults().Query2URL, "/ws/fundamentals-timeseries/v1/finance/timeseries/"+url.PathEscape(symbol), q, &resp)
	return resp, err
}

// FundamentalsTimeseries fetches raw Yahoo fundamentals timeseries types for this ticker.
func (t *Ticker) FundamentalsTimeseries(ctx context.Context, types []string, start, end time.Time) (map[string]any, error) {
	return t.c().FundamentalsTimeseries(ctx, t.Symbol, types, start, end)
}

// SharesFull fetches shares-out timeseries data.
func (c *Client) SharesFull(ctx context.Context, symbol string, start, end time.Time) (map[string]any, error) {
	return c.FundamentalsTimeseries(ctx, symbol, []string{"shares_out"}, start, end)
}

// SharesFull fetches shares-out timeseries data for this ticker.
func (t *Ticker) SharesFull(ctx context.Context, start, end time.Time) (map[string]any, error) {
	return t.c().SharesFull(ctx, t.Symbol, start, end)
}

// IncomeStatement fetches a common income-statement fundamentals timeseries set.
func (c *Client) IncomeStatement(ctx context.Context, symbol string, freq string) (map[string]any, error) {
	return c.FundamentalsTimeseries(ctx, symbol, prefixTimeseriesTypes(freq, incomeStatementTypes), time.Time{}, time.Time{})
}

// BalanceSheet fetches a common balance-sheet fundamentals timeseries set.
func (c *Client) BalanceSheet(ctx context.Context, symbol string, freq string) (map[string]any, error) {
	return c.FundamentalsTimeseries(ctx, symbol, prefixTimeseriesTypes(freq, balanceSheetTypes), time.Time{}, time.Time{})
}

// CashFlow fetches a common cash-flow fundamentals timeseries set.
func (c *Client) CashFlow(ctx context.Context, symbol string, freq string) (map[string]any, error) {
	return c.FundamentalsTimeseries(ctx, symbol, prefixTimeseriesTypes(freq, cashFlowTypes), time.Time{}, time.Time{})
}

func (t *Ticker) IncomeStatement(ctx context.Context, freq string) (map[string]any, error) {
	return t.c().IncomeStatement(ctx, t.Symbol, freq)
}

func (t *Ticker) BalanceSheet(ctx context.Context, freq string) (map[string]any, error) {
	return t.c().BalanceSheet(ctx, t.Symbol, freq)
}

func (t *Ticker) CashFlow(ctx context.Context, freq string) (map[string]any, error) {
	return t.c().CashFlow(ctx, t.Symbol, freq)
}

func prefixTimeseriesTypes(freq string, keys []string) []string {
	prefix := map[string]string{
		"":          "annual",
		"yearly":    "annual",
		"annual":    "annual",
		"quarterly": "quarterly",
		"trailing":  "trailing",
		"ttm":       "trailing",
	}[strings.ToLower(freq)]
	if prefix == "" {
		prefix = freq
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, prefix+key)
	}
	return out
}

var incomeStatementTypes = []string{"TotalRevenue", "CostOfRevenue", "GrossProfit", "OperatingExpense", "OperatingIncome", "EBIT", "EBITDA", "NetIncome", "DilutedEPS", "BasicEPS"}
var balanceSheetTypes = []string{"TotalAssets", "TotalLiabilitiesNetMinorityInterest", "StockholdersEquity", "TotalDebt", "NetDebt", "CashAndCashEquivalents", "WorkingCapital", "OrdinarySharesNumber"}
var cashFlowTypes = []string{"OperatingCashFlow", "FreeCashFlow", "CapitalExpenditure", "InvestingCashFlow", "FinancingCashFlow", "EndCashPosition", "RepurchaseOfCapitalStock", "CashDividendsPaid"}

func nestedSlice(root any, path ...string) []any {
	current := root
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[key]
	}
	values, _ := current.([]any)
	return values
}
