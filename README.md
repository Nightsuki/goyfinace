# goyfinace

`goyfinace` is a Go client for the public Yahoo Finance endpoints used by the Python `yfinance` project. It supports the common workflows: historical OHLCV downloads, quote metadata, fast info, symbol search, option chains, financial statement modules, holders, recommendations, and concurrent multi-symbol history downloads.

Yahoo Finance does not provide these endpoints as a supported public API. Treat upstream schema changes and rate limits as expected operational risks.

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

## Testing

```sh
go test ./...
go vet ./...
```

The unit tests use `httptest` and do not call Yahoo Finance.

## License

Apache License 2.0. This project is a Go implementation inspired by the public behavior of `yfinance`; it is not affiliated with Yahoo.

