package yfinance

import (
	"context"
	"sync"
)

// DownloadResult contains one symbol's download outcome.
type DownloadResult struct {
	Symbol string
	Result *HistoryResult
	Err    error
}

// Download downloads history for multiple symbols concurrently.
func (c *Client) Download(ctx context.Context, symbols []string, params HistoryParams) []DownloadResult {
	results := make([]DownloadResult, len(symbols))
	var wg sync.WaitGroup
	for i, symbol := range symbols {
		i, symbol := i, normalizeSymbol(symbol)
		results[i].Symbol = symbol
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := c.History(ctx, symbol, params)
			results[i].Result = result
			results[i].Err = err
		}()
	}
	wg.Wait()
	return results
}
