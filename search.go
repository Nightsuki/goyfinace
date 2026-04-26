package yfinance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchResult is one symbol returned by Yahoo search.
type SearchResult struct {
	Symbol         string `json:"symbol"`
	ShortName      string `json:"shortname"`
	LongName       string `json:"longname"`
	QuoteType      string `json:"quoteType"`
	Exchange       string `json:"exchange"`
	Score          int    `json:"score"`
	TypeDisp       string `json:"typeDisp"`
	ExchangeDisp   string `json:"exchDisp"`
	Sector         string `json:"sector"`
	Industry       string `json:"industry"`
	IsYahooFinance bool   `json:"isYahooFinance"`
}

// SearchResponse contains Yahoo Finance search results.
type SearchResponse struct {
	Quotes []SearchResult `json:"quotes"`
	News   []any          `json:"news"`
	Raw    map[string]any `json:"-"`
}

// Search queries Yahoo Finance's symbol search endpoint.
func (c *Client) Search(ctx context.Context, query string, quotesCount, newsCount int) (*SearchResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("yfinance: empty search query")
	}
	if quotesCount <= 0 {
		quotesCount = 10
	}
	if newsCount < 0 {
		newsCount = 0
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("quotesCount", strconv.Itoa(quotesCount))
	q.Set("newsCount", strconv.Itoa(newsCount))

	var raw map[string]any
	if err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v1/finance/search", q, &raw); err != nil {
		return nil, err
	}
	var out SearchResponse
	out.Raw = raw
	out.News, _ = raw["news"].([]any)
	if quotes, ok := raw["quotes"].([]any); ok {
		out.Quotes = make([]SearchResult, 0, len(quotes))
		for _, item := range quotes {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out.Quotes = append(out.Quotes, SearchResult{
				Symbol:         stringValue(obj["symbol"]),
				ShortName:      stringValue(obj["shortname"]),
				LongName:       stringValue(obj["longname"]),
				QuoteType:      stringValue(obj["quoteType"]),
				Exchange:       stringValue(obj["exchange"]),
				Score:          int(numberValue(obj["score"])),
				TypeDisp:       stringValue(obj["typeDisp"]),
				ExchangeDisp:   stringValue(obj["exchDisp"]),
				Sector:         stringValue(obj["sector"]),
				Industry:       stringValue(obj["industry"]),
				IsYahooFinance: boolValue(obj["isYahooFinance"]),
			})
		}
	}
	return &out, nil
}
