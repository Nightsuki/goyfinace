package yfinance

import "context"

// HistoryMetadata fetches chart metadata for a symbol.
func (c *Client) HistoryMetadata(ctx context.Context, symbol string) (map[string]any, error) {
	history, err := c.History(ctx, symbol, HistoryParams{Period: Period5D, Interval: Interval1D})
	if err != nil {
		return nil, err
	}
	return history.Meta.Raw, nil
}

// HistoryMetadata fetches chart metadata for this ticker.
func (t *Ticker) HistoryMetadata(ctx context.Context) (map[string]any, error) {
	return t.c().HistoryMetadata(ctx, t.Symbol)
}
