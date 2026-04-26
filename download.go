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
//
// When params.Threads > 0, concurrency is bounded to that many in-flight
// requests. Otherwise every symbol is fetched in its own goroutine.
func (c *Client) Download(ctx context.Context, symbols []string, params HistoryParams) []DownloadResult {
	results := make([]DownloadResult, len(symbols))
	var wg sync.WaitGroup
	var sem chan struct{}
	if params.Threads > 0 {
		sem = make(chan struct{}, params.Threads)
	}
	total := len(symbols)
	for i, symbol := range symbols {
		i, symbol := i, normalizeSymbol(symbol)
		results[i].Symbol = symbol
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}
			result, err := c.History(ctx, symbol, params)
			results[i].Result = result
			results[i].Err = err
			if params.OnProgress != nil {
				params.OnProgress(symbol, i, total, err)
			}
		}()
	}
	wg.Wait()
	return results
}
