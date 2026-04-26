// Package yfinance provides a Go client for public Yahoo Finance endpoints.
//
// The package mirrors the common yfinance workflows for downloading historical
// prices, reading quote metadata, searching symbols, fetching option chains, and
// retrieving quote-summary financial modules. Yahoo Finance does not publish a
// supported public API for these endpoints, so callers should handle upstream
// response changes and rate limiting as normal operational concerns.
//
// A Client is safe to reuse across requests. Create one with NewClient, then
// call methods directly with a symbol:
//
//	client := yfinance.NewClient(nil)
//	history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{
//		Period:   yfinance.Period1Mo,
//		Interval: yfinance.Interval1D,
//	})
//
// For repeated calls against one symbol, use Ticker:
//
//	ticker := client.Ticker("MSFT")
//	info, err := ticker.Info(ctx)
//	options, err := ticker.Options(ctx, time.Time{})
//
// History accepts either a Yahoo period such as Period1Mo or explicit Start and
// End values. When Start or End is set, the client sends Unix period1/period2
// parameters and does not send range. End is exclusive, matching yfinance and
// Yahoo chart behavior.
//
// Methods return normal Go errors. ErrRateLimited maps HTTP 429, ErrNoResult
// indicates a successful response without result data, and YahooError preserves
// structured errors returned by Yahoo JSON payloads.
package yfinance
