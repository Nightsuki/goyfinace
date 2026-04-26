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

// Lists returns the "lists" block from a Yahoo search response. Mirrors
// yfinance's Search.lists.
func (r *SearchResponse) Lists() []map[string]any {
	if r == nil {
		return nil
	}
	return rowsAt(r.Raw, "lists")
}

// Research returns the "researchReports" block from a Yahoo search response.
// Mirrors yfinance's Search.research.
func (r *SearchResponse) Research() []map[string]any {
	if r == nil {
		return nil
	}
	return rowsAt(r.Raw, "researchReports")
}

// NewsRows returns the news block as typed rows when Yahoo returns a list of
// objects. Mirrors yfinance's Search.news.
func (r *SearchResponse) NewsRows() []map[string]any {
	if r == nil {
		return nil
	}
	return rowsAt(r.Raw, "news")
}

// All collects every primary section into a single map keyed by section name
// ("quotes", "news", "lists", "researchReports"). Mirrors yfinance's Search.all.
func (r *SearchResponse) All() map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"quotes":          r.Quotes,
		"news":            r.NewsRows(),
		"lists":           r.Lists(),
		"researchReports": r.Research(),
	}
}

// SearchOptions exposes the optional Yahoo search flags. yfinance forwards
// these as query parameters; the zero value reproduces the historical
// `Search(query, quotesCount, newsCount)` behavior.
type SearchOptions struct {
	QuotesCount                int
	NewsCount                  int
	ListsCount                 int
	EnableFuzzyQuery           *bool
	EnableEnhancedTrivialQuery *bool
	EnablePrivateCompany       *bool
	EnableNavLinks             *bool
	EnableResearchReports      *bool
	EnableCulturalAssets       *bool
	RecommendCount             int
	Region                     string
	Lang                       string
}

// SearchWithOptions runs Yahoo Finance's search endpoint with the full set of
// supported options. Use Search for the simple two-argument form.
func (c *Client) SearchWithOptions(ctx context.Context, query string, opts SearchOptions) (*SearchResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("yfinance: empty search query")
	}
	if opts.QuotesCount <= 0 {
		opts.QuotesCount = 10
	}
	if opts.NewsCount < 0 {
		opts.NewsCount = 0
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("quotesCount", strconv.Itoa(opts.QuotesCount))
	q.Set("newsCount", strconv.Itoa(opts.NewsCount))
	if opts.ListsCount > 0 {
		q.Set("listsCount", strconv.Itoa(opts.ListsCount))
	}
	if opts.RecommendCount > 0 {
		q.Set("recommendCount", strconv.Itoa(opts.RecommendCount))
	}
	if opts.EnableFuzzyQuery != nil {
		q.Set("enableFuzzyQuery", boolFlag(*opts.EnableFuzzyQuery))
	}
	if opts.EnableEnhancedTrivialQuery != nil {
		q.Set("enableEnhancedTrivialQuery", boolFlag(*opts.EnableEnhancedTrivialQuery))
	}
	if opts.EnablePrivateCompany != nil {
		q.Set("enablePrivateCompany", boolFlag(*opts.EnablePrivateCompany))
	}
	if opts.EnableNavLinks != nil {
		q.Set("enableNavLinks", boolFlag(*opts.EnableNavLinks))
	}
	if opts.EnableResearchReports != nil {
		q.Set("enableResearchReports", boolFlag(*opts.EnableResearchReports))
	}
	if opts.EnableCulturalAssets != nil {
		q.Set("enableCulturalAssets", boolFlag(*opts.EnableCulturalAssets))
	}
	if opts.Region != "" {
		q.Set("region", opts.Region)
	}
	if opts.Lang != "" {
		q.Set("lang", opts.Lang)
	}
	return c.runSearch(ctx, q)
}

func boolFlag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// Search queries Yahoo Finance's symbol search endpoint.
func (c *Client) Search(ctx context.Context, query string, quotesCount, newsCount int) (*SearchResponse, error) {
	return c.SearchWithOptions(ctx, query, SearchOptions{
		QuotesCount: quotesCount,
		NewsCount:   newsCount,
	})
}

func (c *Client) runSearch(ctx context.Context, q url.Values) (*SearchResponse, error) {

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
