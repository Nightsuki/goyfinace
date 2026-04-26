# goyfinace

`goyfinace` is a Go client for the public Yahoo Finance endpoints used by the Python `yfinance` project. It supports the common workflows: historical OHLCV downloads, quote metadata, fast info, symbol search, option chains, financial statement modules, holders, recommendations, and concurrent multi-symbol history downloads.

Yahoo Finance does not provide these endpoints as a supported public API. Treat upstream schema changes and rate limits as expected operational risks.

## Acknowledgements

This project is inspired by and API-compatible in spirit with [ranaroussi/yfinance](https://github.com/ranaroussi/yfinance). Thanks to the `yfinance` project for documenting the practical Yahoo Finance workflows that this Go implementation follows.

## Install

```sh
go get github.com/Nightsuki/goyfinace
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	yfinance "github.com/Nightsuki/goyfinace"
)

func main() {
	client := yfinance.NewClient(nil)

	history, err := client.Ticker("AAPL").History(context.Background(), yfinance.HistoryParams{
		Period:   yfinance.Period1Mo,
		Interval: yfinance.Interval1D,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(history.Symbol, len(history.Candles))

	info, err := client.Ticker("AAPL").Info(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info["regularMarketPrice"])
}
```

## API Surface

- `Client.History` / `Ticker.History`: chart candles from `/v8/finance/chart/{symbol}`.
- `Client.Download`: concurrent history downloads for multiple symbols.
- `Client.Info` / `Ticker.Info`: common `quoteSummary` modules flattened into a map.
- `Client.FastInfo`: selected price metadata from chart metadata.
- `Client.QuoteSummary`: raw `quoteSummary` module access for advanced callers.
- `Client.Search`: symbol lookup via Yahoo Finance search.
- `Client.Options`: option chains for the default or requested expiration.
- `Client.Financials`, `Client.Holders`, `Client.Recommendations`: convenience wrappers over quote-summary modules.
- yfinance-compatible extras: `Actions`, `Dividends`, `Splits`, `CapitalGains`, `Quote`, `News`, `Calendar`, `SECFilings`, `Sustainability`, `Valuation`, `Analysis`, analyst estimates, holder/insider helpers, `FundProfile`, `FundsData`, `SharesFull`, `Shares`, statement timeseries, `Lookup`, `PredefinedScreen`, `Screen`, screener query builders, `Market`, `Tickers`, `Calendars`, `MarketSummary`, `MarketStatus`, `Sector`, `Industry`, typed `SectorOf`/`IndustryOf` accessors, `EarningsDates`, and `WebSocket`.
- Pricing adjustments: `HistoryParams.AutoAdjust`, `BackAdjust`, `Rounding`, and `Repair` mirror yfinance's `auto_adjust`/`back_adjust`/`rounding`/`repair` flags. `HistoryParams.Threads` bounds `Download` concurrency.
- Cookie + crumb authentication: `Client.Authenticate(ctx)` primes the jar and crumb; subsequent calls to crumb-protected endpoints attach `crumb` automatically.
- Reliability & observability: `Client.Logger` (slog), `Client.Retries` + `Client.RetryBackoff` (exponential backoff for 5xx/rate-limit/transient network errors), and `Client.Limiter` (`NewRateLimiter(rps, burst)` or any `golang.org/x/time/rate.Limiter`).
- Download progress callbacks via `HistoryParams.OnProgress`; `HistoryParams.NoEvents` explicitly suppresses Yahoo events; `HistoryParams.DropNaN` drops all-NaN rows.
- Advanced search via `Client.SearchWithOptions` exposing fuzzy / enhanced-trivial / private-company / region / lang flags.
- Optional GET-JSON response cache: `Client.Cache` (`NewMemoryCache()` or any `Cache` implementation) with `Client.CacheTTL`.
- Auto-reconnecting `WebSocket`: `AutoReconnect`, `ReconnectBackoff`, `MaxReconnectAttempts`, `OnReconnect` with transparent re-subscribe on reconnection.

The default `UserAgent` mirrors the browser user-agent list used by upstream `yfinance`. Set `Client.UserAgent` when a fixed application-specific value is required.

Detailed API documentation is available in English by default, with Chinese as a localized version:

- English: [docs/API.md](docs/API.md)
- Chinese: [docs/API.zh.md](docs/API.zh.md)

## Date Ranges

Use either a Yahoo period string or explicit start/end timestamps:

```go
history, err := client.History(ctx, "MSFT", yfinance.HistoryParams{
	Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	End:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	Interval: yfinance.Interval1D,
})
```

Yahoo treats `End` as exclusive, matching `yfinance` behavior.

## DataFrame Adapter

The core API returns Go structs, slices, and maps instead of exposing a DataFrame type. For pandas-style table workflows, import the Gota adapter:

```go
import yfgota "github.com/Nightsuki/goyfinace/adapter/gota"

history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{
	Period:   yfinance.Period1Mo,
	Interval: yfinance.Interval1D,
})
if err != nil {
	log.Fatal(err)
}

df := yfgota.History(history)
fmt.Println(df.Nrow(), df.Col("Close").Float())
```

The adapter currently supports history candles, corporate actions, option contracts/chains, search results, quote-summary modules, fundamentals timeseries, key/value maps, and generic records.

## Testing

```sh
go test ./...
go vet ./...
```

The unit tests use `httptest` and do not call Yahoo Finance.

## License

Apache License 2.0. This project is a Go implementation inspired by the public behavior of `yfinance`; it is not affiliated with Yahoo.
