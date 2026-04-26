package yfinance

import (
	"context"
	"net/url"
	"strings"
)

// ISIN attempts to resolve a ticker's ISIN through an external suggestion
// endpoint. Yahoo does not expose a stable ticker-to-ISIN endpoint, so this
// helper is best-effort and returns ErrNoResult when it cannot verify an exact
// symbol match.
func (c *Client) ISIN(ctx context.Context, symbol string) (string, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" || strings.ContainsAny(symbol, "-^") {
		return "", ErrNoResult
	}
	info, err := c.Info(ctx, symbol)
	if err != nil {
		return "", err
	}
	query := symbol
	if shortName, ok := info["shortName"].(string); ok && shortName != "" {
		query = shortName
	}
	q := url.Values{}
	q.Set("max_results", "25")
	q.Set("query", query)
	text, err := c.getText(ctx, c.cloneWithDefaults().ISINURL, "/ajax/SearchController_Suggest", q)
	if err != nil {
		return "", err
	}
	isin, ok := parseISINSuggestion(text, symbol)
	if !ok {
		return "", ErrNoResult
	}
	return isin, nil
}

func parseISINSuggestion(text string, symbol string) (string, bool) {
	search := `"` + symbol + `|`
	start := 0
	for {
		idx := strings.Index(text[start:], search)
		if idx < 0 {
			return "", false
		}
		idx += start + len(search)
		record := strings.SplitN(text[idx:], `"`, 2)[0]
		isin := strings.SplitN(record, "|", 2)[0]
		if isin != "" {
			return isin, true
		}
		start = idx
	}
}

// ISIN attempts to resolve this ticker's ISIN.
func (t *Ticker) ISIN(ctx context.Context) (string, error) {
	return t.c().ISIN(ctx, t.Symbol)
}
