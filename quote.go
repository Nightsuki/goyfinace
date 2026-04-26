package yfinance

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

var defaultInfoModules = []string{
	"price",
	"summaryDetail",
	"financialData",
	"quoteType",
	"defaultKeyStatistics",
	"assetProfile",
}

// QuoteSummary fetches raw quoteSummary modules for a symbol.
func (t *Ticker) QuoteSummary(ctx context.Context, modules ...string) (map[string]any, error) {
	return t.c().QuoteSummary(ctx, t.Symbol, modules...)
}

// Info fetches the common yfinance info module set and flattens one level.
func (t *Ticker) Info(ctx context.Context) (map[string]any, error) {
	return t.c().Info(ctx, t.Symbol)
}

// FastInfo returns selected price metadata from the chart endpoint.
func (t *Ticker) FastInfo(ctx context.Context) (*FastInfo, error) {
	return t.c().FastInfo(ctx, t.Symbol)
}

// QuoteSummary fetches raw quoteSummary modules for a symbol.
func (c *Client) QuoteSummary(ctx context.Context, symbol string, modules ...string) (map[string]any, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("yfinance: empty symbol")
	}
	if len(modules) == 0 {
		modules = defaultInfoModules
	}
	q := url.Values{}
	q.Set("modules", strings.Join(modules, ","))
	q.Set("corsDomain", "finance.yahoo.com")
	q.Set("formatted", "false")

	var resp quoteSummaryResponse
	err := c.getJSON(ctx, c.cloneWithDefaults().Query2URL, "/v10/finance/quoteSummary/"+url.PathEscape(symbol), q, &resp)
	if err != nil {
		return nil, err
	}
	if resp.QuoteSummary.Error != nil {
		return nil, resp.QuoteSummary.Error
	}
	if len(resp.QuoteSummary.Result) == 0 {
		return nil, ErrNoResult
	}
	return resp.QuoteSummary.Result[0], nil
}

// Info fetches the common yfinance info module set and flattens one level.
func (c *Client) Info(ctx context.Context, symbol string) (map[string]any, error) {
	modules, err := c.QuoteSummary(ctx, symbol, defaultInfoModules...)
	if err != nil {
		return nil, err
	}
	flat := make(map[string]any)
	for _, module := range modules {
		obj, ok := module.(map[string]any)
		if !ok {
			continue
		}
		for key, value := range obj {
			flat[key] = unwrapYahooValue(value)
		}
	}
	return flat, nil
}

// FastInfo contains selected ticker metadata.
type FastInfo struct {
	Symbol           string
	Currency         string
	ExchangeName     string
	FullExchangeName string
	InstrumentType   string
	Timezone         string
	// ExchangeTimezoneName is the IANA timezone name returned in chart metadata.
	ExchangeTimezoneName string
	RegularMarketPrice   float64
	PreviousClose        float64
}

// FastInfo returns selected price metadata from the chart endpoint.
func (c *Client) FastInfo(ctx context.Context, symbol string) (*FastInfo, error) {
	history, err := c.History(ctx, symbol, HistoryParams{Period: Period5D, Interval: Interval1D})
	if err != nil {
		return nil, err
	}
	meta := history.Meta
	return &FastInfo{
		Symbol:               meta.Symbol,
		Currency:             meta.Currency,
		ExchangeName:         meta.ExchangeName,
		FullExchangeName:     meta.FullExchangeName,
		InstrumentType:       meta.InstrumentType,
		Timezone:             meta.Timezone,
		ExchangeTimezoneName: meta.ExchangeTimezoneName,
		RegularMarketPrice:   meta.RegularMarketPrice,
		PreviousClose:        meta.ChartPreviousClose,
	}, nil
}
