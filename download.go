package yfinance

import (
	"context"
	"sync"
)

// DownloadResult contains one symbol's download outcome.
type DownloadResult struct {
	// Symbol is the normalized uppercase symbol requested for this slot.
	Symbol string
	// Result is populated when Err is nil.
	Result *HistoryResult
	// Err is the per-symbol error. Download does not fail fast; inspect each
	// result independently.
	Err error
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
