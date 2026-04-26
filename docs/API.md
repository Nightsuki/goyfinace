# goyfinace API Documentation

Chinese localization: [API.zh.md](API.zh.md)

This document describes the public API of `github.com/Nightsuki/goyfinace`: entry points, parameters, return values, and error handling. The package name is `yfinance`, so using an explicit import alias is recommended:

```go
import yfinance "github.com/Nightsuki/goyfinace"
```

## Scope

`goyfinace` is a Go client for the public Yahoo Finance endpoints used by the Python `yfinance` project. It covers the common workflows:

- Historical OHLCV data: `History`
- Concurrent multi-symbol downloads: `Download`
- Common quote metadata: `Info`
- Lightweight price metadata: `FastInfo`
- Raw quote summary modules: `QuoteSummary`
- Option chains: `Options`
- Symbol search: `Search`
- Financials, holders, and recommendations: `Financials`, `Holders`, `Recommendations`

Yahoo Finance does not publish these endpoints as a supported public API. Production callers should handle rate limits, missing fields, upstream schema changes, and ordinary network failures.

## Installation

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
	ctx := context.Background()
	client := yfinance.NewClient(nil)

	history, err := client.Ticker("AAPL").History(ctx, yfinance.HistoryParams{
		Period:   yfinance.Period1Mo,
		Interval: yfinance.Interval1D,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(history.Symbol, len(history.Candles))

	info, err := client.Ticker("AAPL").Info(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info["regularMarketPrice"])
}
```

## Client

### NewClient

```go
func NewClient(httpClient *http.Client) *Client
```

Creates a Yahoo Finance client. Passing `nil` creates a default `http.Client` with a 15 second timeout.

### Client Fields

```go
type Client struct {
	HTTPClient *http.Client
	Query1URL  string
	Query2URL  string
	RootURL    string
	ISINURL    string
	UserAgent  string
}
```

| Field | Description |
| --- | --- |
| `HTTPClient` | HTTP client used for all requests. Use it for custom timeouts, proxies, transports, or retry wrappers. |
| `Query1URL` | Base URL for `query1.finance.yahoo.com`. Override for tests or internal proxies. |
| `Query2URL` | Base URL for `query2.finance.yahoo.com`. Override for tests or internal proxies. |
| `RootURL` | Base URL for Yahoo Finance frontend JSON endpoints such as news. |
| `ISINURL` | External best-effort ISIN suggestion endpoint. Keep separate from Yahoo endpoints because Yahoo does not expose stable ISIN lookup. |
| `UserAgent` | User-Agent sent on each request. Empty uses the package default browser user-agent from `DefaultUserAgents`. |

`Client` can be reused across requests. In most applications, create one client at startup and share it.

### Ticker

```go
func (c *Client) Ticker(symbol string) *Ticker
```

Creates a symbol-scoped helper. The symbol is trimmed and uppercased:

```go
ticker := client.Ticker("msft")
info, err := ticker.Info(ctx)
history, err := ticker.History(ctx, yfinance.HistoryParams{Period: yfinance.Period5D})
```

Use `Client` methods for direct one-off calls and `Ticker` methods when several calls target the same symbol.

## Historical Data: History

### Methods

```go
func (c *Client) History(ctx context.Context, symbol string, params HistoryParams) (*HistoryResult, error)
func (t *Ticker) History(ctx context.Context, params HistoryParams) (*HistoryResult, error)
```

Downloads OHLCV candles from Yahoo's chart endpoint.

### HistoryParams

```go
type HistoryParams struct {
	Period     string
	Interval   string
	Start      time.Time
	End        time.Time
	PrePost    bool
	Events     []string
	AutoAdjust bool
	BackAdjust bool
	Rounding   bool
	Repair     bool
	Threads    int
}
```

| Field | Description |
| --- | --- |
| `Period` | Yahoo range value. Used only when both `Start` and `End` are zero. Defaults to `Period1Mo`. |
| `Interval` | Candle interval. Defaults to `Interval1D`. |
| `Start` | Inclusive start time for an explicit date range. Setting `Start` or `End` uses `period1/period2`. |
| `End` | Exclusive end time for an explicit date range. If `Start` is set and `End` is zero, the current time is used. |
| `PrePost` | Requests pre-market and post-market rows when Yahoo supports them. |
| `Events` | Event types to request. Empty requests dividends, splits, and capital gains. |
| `AutoAdjust` | Applies `AdjClose/Close` ratio to OHLC and inverse to Volume so OHLC are split- and dividend-adjusted. Mirrors yfinance's `auto_adjust=True`. |
| `BackAdjust` | Back-adjusts older candles so the most recent Close matches AdjClose, preserving the latest raw price. Mirrors yfinance's `back_adjust=True`. |
| `Rounding` | Rounds OHLC values to the chart's `PriceHint` decimals. Mirrors yfinance's `rounding=True`. |
| `Repair` | Heuristically fixes 100x and 0.01x price anomalies (Yahoo currency-unit changes) by comparing each row's Close to its neighborhood median. Mirrors yfinance's `repair=True`. |
| `Threads` | Bounds in-flight Download concurrency. Zero or negative means unlimited. Ignored by `History`. Mirrors yfinance's `download(threads=N)`. |
| `NoEvents` | Suppresses dividend/split/capital-gain event requests. By default Yahoo events are included. |
| `OnProgress` | Optional `func(symbol, idx, total int, err error)` invoked by `Download` once per symbol after completion. Mirrors yfinance's tqdm-driven progress hook. |
| `DropNaN` | Drops candles whose every OHLC value is NaN. Mirrors yfinance's `dropna=True`. |

### Period Constants

```go
Period1D  = "1d"
Period5D  = "5d"
Period1Mo = "1mo"
Period3Mo = "3mo"
Period6Mo = "6mo"
Period1Y  = "1y"
Period2Y  = "2y"
Period5Y  = "5y"
Period10Y = "10y"
PeriodYTD = "ytd"
PeriodMax = "max"
```

### Interval Constants

```go
Interval1M  = "1m"
Interval2M  = "2m"
Interval5M  = "5m"
Interval15M = "15m"
Interval30M = "30m"
Interval60M = "60m"
Interval90M = "90m"
Interval1H  = "1h"
Interval1D  = "1d"
Interval5D  = "5d"
Interval1Wk = "1wk"
Interval1Mo = "1mo"
Interval3Mo = "3mo"
```

Yahoo usually limits the historical range available for intraday intervals. Requests outside those limits may return errors or empty results.

### HistoryResult

```go
type HistoryResult struct {
	Symbol   string
	Meta     ChartMeta
	Candles  []Candle
	YahooErr *YahooError
}
```

| Field | Description |
| --- | --- |
| `Symbol` | Normalized symbol. |
| `Meta` | Selected metadata from Yahoo chart. |
| `Candles` | OHLCV rows. |
| `YahooErr` | Structured Yahoo error when one is returned. The same error is also returned as `error`. |

### Candle

```go
type Candle struct {
	Time      time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	AdjClose  float64
	Volume    int64
	Dividends float64
	Split     float64
}
```

| Field | Description |
| --- | --- |
| `Time` | UTC candle timestamp. |
| `Open`, `High`, `Low`, `Close` | Raw OHLC prices. Yahoo `null` values become `math.NaN()`. |
| `AdjClose` | Adjusted close. If Yahoo omits it, `Close` is used. |
| `Volume` | Reported volume. Missing volume is zero. |
| `Dividends` | Dividend amount at this timestamp, or zero. |
| `Split` | Split ratio at this timestamp, for example `4` for a 4-for-1 split. |

### ChartMeta

```go
type ChartMeta struct {
	Currency             string
	Symbol               string
	ExchangeName         string
	FullExchangeName     string
	InstrumentType       string
	FirstTradeDate       time.Time
	RegularMarketTime    time.Time
	GMTOffset            int
	Timezone             string
	ExchangeTimezoneName string
	RegularMarketPrice   float64
	ChartPreviousClose   float64
	PriceHint            int
	Raw                  map[string]any
}
```

`Raw` preserves the full Yahoo metadata object for fields that are not promoted into the struct.

### Example: Range by Period

```go
history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{
	Period:   yfinance.Period6Mo,
	Interval: yfinance.Interval1D,
})
if err != nil {
	return err
}
for _, row := range history.Candles {
	fmt.Println(row.Time, row.Close, row.Volume)
}
```

### Example: Explicit Date Range

```go
history, err := client.History(ctx, "MSFT", yfinance.HistoryParams{
	Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	End:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	Interval: yfinance.Interval1D,
})
```

`End` is exclusive, matching Python `yfinance` and Yahoo chart behavior.

## Multi-Symbol Downloads: Download

```go
func (c *Client) Download(ctx context.Context, symbols []string, params HistoryParams) []DownloadResult
```

Downloads history for several symbols concurrently. The returned slice preserves the input order.

```go
results := client.Download(ctx, []string{"AAPL", "MSFT", "GOOG"}, yfinance.HistoryParams{
	Period:   yfinance.Period5D,
	Interval: yfinance.Interval1D,
})

for _, item := range results {
	if item.Err != nil {
		fmt.Println(item.Symbol, item.Err)
		continue
	}
	fmt.Println(item.Symbol, len(item.Result.Candles))
}
```

`Download` does not fail fast. Inspect each `DownloadResult.Err` independently.

```go
type DownloadResult struct {
	Symbol string
	Result *HistoryResult
	Err    error
}
```

## DataFrame Adapter: Gota

The root package intentionally returns Go-native values and does not expose a DataFrame type. For users migrating from pandas-oriented `yfinance` workflows, `goyfinace` provides an optional adapter package:

```go
import yfgota "github.com/Nightsuki/goyfinace/adapter/gota"
```

Because the adapter lives in a separate import path, applications that only need the core HTTP client do not need to use DataFrame APIs directly.

### History DataFrames

```go
history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{
	Period:   yfinance.Period1Mo,
	Interval: yfinance.Interval1D,
})
if err != nil {
	return err
}

df := yfgota.History(history)
fmt.Println(df.Nrow())
fmt.Println(df.Col("Close").Float())
```

`yfgota.History` and `yfgota.Candles` produce columns that are familiar to Python `yfinance` users:

- `Time`
- `Timestamp`
- `Open`
- `High`
- `Low`
- `Close`
- `Adj Close`
- `Volume`
- `Dividends`
- `Stock Splits`

`Time` is formatted as RFC3339 UTC text. `Timestamp` stores Unix seconds for callers that prefer numeric sorting or filtering.

### Option DataFrames

```go
chain, err := client.Options(ctx, "AAPL", time.Time{})
if err != nil {
	return err
}

calls, puts := yfgota.Options(chain)
fmt.Println(calls.Nrow(), puts.Nrow())
```

You can also convert one side directly:

```go
calls := yfgota.OptionContracts(chain.Calls)
```

Option DataFrame columns follow Yahoo option-chain field names such as `contractSymbol`, `lastTradeDate`, `strike`, `lastPrice`, `bid`, `ask`, `volume`, `openInterest`, and `impliedVolatility`.

### Search and QuoteSummary DataFrames

```go
search, err := client.Search(ctx, "apple", 10, 0)
if err != nil {
	return err
}
quotes := yfgota.Search(search)

summary, err := client.QuoteSummary(ctx, "AAPL", "price", "summaryDetail")
if err != nil {
	return err
}
longForm := yfgota.QuoteSummary(summary)
```

`QuoteSummary` returns a long-form table with `Module`, `Key`, and `Value` columns. Yahoo `{raw, fmt}` values are unwrapped to `raw` first, and nested values are JSON stringified so they can live in scalar DataFrame cells.

### Generic Map and Record Helpers

```go
kv := yfgota.KeyValues(info)
records := yfgota.Records([]map[string]any{
	{"symbol": "AAPL", "regularMarketPrice": 201.5},
})
```

Use these helpers for financial modules, holders, recommendations, or your own normalized records.

## QuoteSummary

```go
func (c *Client) QuoteSummary(ctx context.Context, symbol string, modules ...string) (map[string]any, error)
func (t *Ticker) QuoteSummary(ctx context.Context, modules ...string) (map[string]any, error)
```

Reads raw Yahoo quoteSummary modules. Use this when you need fields that are not wrapped by convenience methods.

```go
summary, err := client.QuoteSummary(ctx, "AAPL", "price", "summaryDetail")
if err != nil {
	return err
}
price := summary["price"].(map[string]any)
fmt.Println(price["symbol"])
```

When `modules` is empty, the default module set used by `Info` is requested:

- `price`
- `summaryDetail`
- `financialData`
- `quoteType`
- `defaultKeyStatistics`
- `assetProfile`

The return value is a raw Yahoo module map. Since Yahoo fields can change, protect type assertions in production code.

## Info

```go
func (c *Client) Info(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Info(ctx context.Context) (map[string]any, error)
```

Requests common quoteSummary modules and flattens one level into `map[string]any`. Yahoo `{raw, fmt}` value objects are unwrapped to `raw` first, then `fmt` when `raw` is absent.

```go
info, err := client.Info(ctx, "AAPL")
if err != nil {
	return err
}

fmt.Println(info["symbol"])
fmt.Println(info["regularMarketPrice"])
fmt.Println(info["marketCap"])
```

Use `Info` for quick metadata reads similar to Python `Ticker.info`. For strict schemas, prefer `QuoteSummary` with explicit modules and decode into application-owned structs.

## FastInfo

```go
func (c *Client) FastInfo(ctx context.Context, symbol string) (*FastInfo, error)
func (t *Ticker) FastInfo(ctx context.Context) (*FastInfo, error)
```

Returns selected price metadata from chart metadata. The current implementation requests `5d/1d` chart metadata.

```go
fast, err := client.FastInfo(ctx, "AAPL")
if err != nil {
	return err
}
fmt.Println(fast.Currency, fast.RegularMarketPrice, fast.PreviousClose)
```

```go
type FastInfo struct {
	Symbol               string
	Currency             string
	ExchangeName         string
	FullExchangeName     string
	InstrumentType       string
	Timezone             string
	ExchangeTimezoneName string
	RegularMarketPrice   float64
	PreviousClose        float64
}
```

## Search

```go
func (c *Client) Search(ctx context.Context, query string, quotesCount, newsCount int) (*SearchResponse, error)
```

Searches Yahoo Finance symbols.

```go
resp, err := client.Search(ctx, "apple", 10, 0)
if err != nil {
	return err
}
for _, quote := range resp.Quotes {
	fmt.Println(quote.Symbol, quote.ShortName, quote.QuoteType)
}
```

| Parameter | Description |
| --- | --- |
| `query` | Search text. Must not be empty. |
| `quotesCount` | Number of quote results. Defaults to 10 when less than or equal to zero. |
| `newsCount` | Number of news results. Negative values are treated as zero. |

```go
type SearchResponse struct {
	Quotes []SearchResult
	News   []any
	Raw    map[string]any
}
```

`Raw` preserves the full Yahoo response.

```go
type SearchResult struct {
	Symbol         string
	ShortName      string
	LongName       string
	QuoteType      string
	Exchange       string
	Score          int
	TypeDisp       string
	ExchangeDisp   string
	Sector         string
	Industry       string
	IsYahooFinance bool
}
```

## Option Chains: Options

```go
func (c *Client) Options(ctx context.Context, symbol string, expiration time.Time) (*OptionChain, error)
func (t *Ticker) Options(ctx context.Context, expiration time.Time) (*OptionChain, error)
```

Fetches an option chain. Pass `time.Time{}` to use Yahoo's default expiration.

```go
chain, err := client.Options(ctx, "AAPL", time.Time{})
if err != nil {
	return err
}

fmt.Println(chain.Symbol, chain.ExpirationDate)
for _, call := range chain.Calls {
	fmt.Println(call.ContractSymbol, call.Strike, call.LastPrice)
}
```

To request a specific expiration, first read `Expirations`, then call `Options` again with one of those dates:

```go
first, err := client.Options(ctx, "AAPL", time.Time{})
if err != nil {
	return err
}
if len(first.Expirations) > 0 {
	chain, err := client.Options(ctx, "AAPL", first.Expirations[0])
	_ = chain
	_ = err
}
```

```go
type OptionChain struct {
	Symbol         string
	Underlying     map[string]any
	ExpirationDate time.Time
	Expirations    []time.Time
	Strikes        []float64
	Calls          []OptionContract
	Puts           []OptionContract
}
```

`Underlying` is Yahoo's raw quote object for the underlying instrument.

```go
type OptionContract struct {
	ContractSymbol    string
	Strike            float64
	Currency          string
	LastPrice         float64
	Change            float64
	PercentChange     float64
	Volume            int64
	OpenInterest      int64
	Bid               float64
	Ask               float64
	ContractSize      string
	Expiration        time.Time
	LastTradeDate     time.Time
	ImpliedVolatility float64
	InTheMoney        bool
}
```

`ImpliedVolatility` is decimal volatility, for example `0.25` for 25%.

## Financials, Holders, and Recommendations

These methods are convenience wrappers over `QuoteSummary` and return raw Yahoo module maps.

### Financials

```go
func (c *Client) Financials(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Financials(ctx context.Context) (map[string]any, error)
```

Requested modules:

- `incomeStatementHistory`
- `incomeStatementHistoryQuarterly`
- `balanceSheetHistory`
- `balanceSheetHistoryQuarterly`
- `cashflowStatementHistory`
- `cashflowStatementHistoryQuarterly`
- `earnings`
- `earningsTrend`

```go
financials, err := client.Financials(ctx, "AAPL")
if err != nil {
	return err
}
fmt.Println(financials["incomeStatementHistory"])
```

### Holders

```go
func (c *Client) Holders(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Holders(ctx context.Context) (map[string]any, error)
```

Requested modules:

- `institutionOwnership`
- `fundOwnership`
- `majorHoldersBreakdown`
- `insiderHolders`

### Recommendations

```go
func (c *Client) Recommendations(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Recommendations(ctx context.Context) (map[string]any, error)
```

Requested modules:

- `recommendationTrend`
- `upgradeDowngradeHistory`

## Error Handling

The package defines two sentinel errors:

```go
var ErrNoResult = errors.New("yfinance: no result")
var ErrRateLimited = errors.New("yfinance: rate limited")
```

Example:

```go
history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{})
if err != nil {
	switch {
	case errors.Is(err, yfinance.ErrRateLimited):
		// Back off or reduce concurrency.
	case errors.Is(err, yfinance.ErrNoResult):
		// Missing symbol, unavailable data, or empty Yahoo result.
	default:
		var yahooErr *yfinance.YahooError
		if errors.As(err, &yahooErr) {
			fmt.Println(yahooErr.Code, yahooErr.Description)
		}
		return err
	}
}
```

`YahooError` preserves structured Yahoo JSON errors:

```go
type YahooError struct {
	Code        string
	Description string
}
```

Non-2xx HTTP responses return ordinary errors containing the status code and a response snippet.

## Additional yfinance Helpers

### Multi-Symbol Tickers

```go
tickers := client.Tickers("AAPL MSFT")
quotes, err := tickers.Quotes(ctx)
downloads := tickers.Download(ctx, yfinance.HistoryParams{Period: yfinance.Period1Mo})
```

`Tickers` stores normalized symbols and delegates to `Quote` and `Download`.

### Screener Query Builders

```go
query := yfinance.EquityQuery(
	yfinance.GTE("intradaymarketcap", 1_000_000_000),
	yfinance.Between("eodvolume", 1_000_000, 10_000_000),
)
result, err := client.Screen(ctx, yfinance.ScreenRequest{Query: query, Size: 50})
```

`EquityQuery`, `FundQuery`, and `ETFQuery` create Yahoo screener query maps. Lower-level builders include `Eq`, `GT`, `GTE`, `LT`, `LTE`, `Between`, `And`, `Or`, and `Screener`.

### Estimate, Holder, and Insider Helpers

The following helpers are thin quoteSummary wrappers and return Yahoo's raw module payloads:

```go
client.EarningsEstimate(ctx, "AAPL")
client.RevenueEstimate(ctx, "AAPL")
client.EarningsHistory(ctx, "AAPL")
client.EPSRevisions(ctx, "AAPL")
client.EPSTrend(ctx, "AAPL")
client.GrowthEstimates(ctx, "AAPL")
client.RecommendationsSummary(ctx, "AAPL")
client.MajorHolders(ctx, "AAPL")
client.InstitutionalHolders(ctx, "AAPL")
client.MutualFundHolders(ctx, "AAPL")
client.InsiderPurchases(ctx, "AAPL")
client.InsiderTransactions(ctx, "AAPL")
client.InsiderRosterHolders(ctx, "AAPL")
```

### Market and Calendars

```go
summary, err := client.Market("us").Summary(ctx)
status, err := client.Market("us").Status(ctx)
earnings, err := client.Calendars().Earnings(ctx, 25)
```

### WebSocket Streaming

```go
ws := yfinance.NewWebSocket("")
defer ws.Close()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := ws.Subscribe(ctx, "AAPL", "MSFT"); err != nil {
	return err
}
err := ws.Listen(ctx, func(msg yfinance.StreamMessage) {
	fmt.Println(msg.ID, msg.Price)
})
```

`NewAsyncWebSocket` and `Client.AsyncWebSocket` return the same context-driven Go client. Incoming Yahoo frames are decoded into `StreamMessage`; the original values remain available in `Raw`.

## Context and Timeouts

All network methods accept `context.Context`. Prefer setting a business-level timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{})
```

You can also pass a custom `http.Client`:

```go
client := yfinance.NewClient(&http.Client{
	Timeout: 5 * time.Second,
})
```

## Testing and Proxies

`Client.Query1URL` and `Client.Query2URL` are useful for tests and internal proxies:

```go
client := yfinance.NewClient(httpClient)
client.Query1URL = "http://127.0.0.1:8080"
client.Query2URL = "http://127.0.0.1:8080"
```

The repository unit tests use `httptest` fixtures and do not call Yahoo Finance.

## Differences from Python yfinance

Main differences:

- The Go package returns structs, slices, and maps instead of DataFrames.
- Optional DataFrame support is available through `github.com/Nightsuki/goyfinace/adapter/gota`; it is not a full pandas clone.
- `Info` returns a flattened map but does not promise a fixed field set.
- Financial, screener, calendar, domain, and analysis methods preserve Yahoo's raw module/endpoint structure for caller-owned modeling.
- Python-specific scraping, `repair` heuristics, full pandas index/MultiIndex behavior, and Python `asyncio` APIs are not replicated in the root package.

## yfinance Compatibility Surface

Beyond the core APIs above, the package exposes Go-native equivalents for the main yfinance feature families:

- Corporate actions: `Actions`, `Dividends`, `Splits`, `CapitalGains`.
- Quote and quote-summary extras: `Quote`, `Calendar`, `SECFilings`, `Sustainability`, `Valuation`.
- Analyst data: `Analysis`, `AnalystPriceTargets`, `UpgradesDowngrades`, `Recommendations`.
- Funds and holders: `FundProfile`, `FundsData`, `Holders`.
- Financial statements and shares: `FundamentalsTimeseries`, `IncomeStatement`, `BalanceSheet`, `CashFlow`, `SharesFull`.
- Estimates, holders, and insiders: `EarningsEstimate`, `RevenueEstimate`, `EarningsHistory`, `EPSRevisions`, `EPSTrend`, `GrowthEstimates`, `RecommendationsSummary`, `MajorHolders`, `InstitutionalHolders`, `MutualFundHolders`, `InsiderPurchases`, `InsiderTransactions`, `InsiderRosterHolders`.
- Discovery and screeners: `Lookup`, `LookupISIN`, `Search`, `Screen`, `PredefinedScreen`, `EquityQuery`, `FundQuery`, `ETFQuery`.
- Market/domain data: `Market`, `MarketSummary`, `MarketStatus`, `Sector`, `Industry`, `SectorOf`, `IndustryOf`.
- Calendars/news: `CalendarVisualization`, `EarningsDates`, `News`.
- Multi-symbol and streaming helpers: `Tickers`, `WebSocket`, `AsyncWebSocket`.

These methods intentionally return raw `map[string]any` payloads or simple records when Yahoo's schema is broad or unstable. Use `adapter/gota` helpers such as `QuoteSummary`, `KeyValues`, `Records`, `Actions`, and `Timeseries` when you want a table representation.

Migration examples:

| Python yfinance | goyfinace |
| --- | --- |
| `yf.Ticker("AAPL").history(...)` | `client.Ticker("AAPL").History(ctx, params)` |
| `yf.download(["AAPL", "MSFT"])` | `client.Download(ctx, []string{"AAPL", "MSFT"}, params)` |
| `yf.Tickers("AAPL MSFT")` | `client.Tickers("AAPL MSFT")` |
| `ticker.info` | `ticker.Info(ctx)` |
| `ticker.fast_info` | `ticker.FastInfo(ctx)` |
| `ticker.option_chain(...)` | `ticker.Options(ctx, expiration)` |
| `ticker.financials` | `ticker.Financials(ctx)` |
| `yf.WebSocket(...)` | `yfinance.NewWebSocket(url)` |
| `yf.Sector("technology").top_companies` | `client.SectorOf(ctx, "technology").TopCompanies()` |
| `yf.Industry("software-application").top_growth_companies` | `client.IndustryOf(ctx, "software-application").TopGrowthCompanies()` |
| `ticker.funds_data.top_holdings` | `ticker.FundsData(ctx).TopHoldings()` |
| `Search(...).lists` | `client.Search(ctx, q, n, m).Lists()` |
| `ticker.history(auto_adjust=True)` | `ticker.History(ctx, HistoryParams{AutoAdjust: true})` |

## Typed Domain Accessors

Beyond the raw `Sector`/`Industry`/`FundProfile` map endpoints, the package exposes typed wrappers that mirror the property surface of yfinance's `Sector`, `Industry`, and `FundsData` classes.

### SectorOf / IndustryOf

```go
sector, err := client.SectorOf(ctx, "technology")
if err != nil {
    return err
}

fmt.Println(sector.Name)
fmt.Println(sector.Overview())
for _, row := range sector.TopCompanies() {
    fmt.Println(row["symbol"])
}
for _, row := range sector.TopETFs() { /* ... */ }
for _, row := range sector.TopMutualFunds() { /* ... */ }
for _, row := range sector.Industries() { /* ... */ }
sector.TopGrowthCompanies()
sector.TopPerformingCompanies()

industry, err := client.IndustryOf(ctx, "software-application")
if err != nil {
    return err
}
fmt.Println(industry.Name, industry.SectorKey, industry.SectorName)
industry.Overview()
industry.TopPerformingCompanies()
industry.TopGrowthCompanies()
industry.KeyCompanyKeys()
industry.KeyCompanyGroups()
```

`SectorData.Raw` and `IndustryData.Raw` preserve the full Yahoo response for fields not promoted by the typed methods.

### FundsData

```go
funds, err := client.Ticker("VGT").FundsData(ctx)
if err != nil {
    return err
}

fmt.Println(funds.Description())
fmt.Println(funds.FundOverview())   // family / category / legalType
fmt.Println(funds.FundOperations()) // expense ratios
fmt.Println(funds.AssetClasses())   // cashPosition, stockPosition, ...
for _, h := range funds.TopHoldings() {
    fmt.Println(h["symbol"], h["holdingPercent"])
}
funds.EquityHoldings()
funds.BondHoldings()
funds.BondRatings()
funds.SectorWeightings()
```

### Search Result Sections

`SearchResponse` exposes typed accessors for every primary section returned by Yahoo:

```go
resp, err := client.Search(ctx, "apple", 10, 5)
if err != nil {
    return err
}

for _, q := range resp.Quotes { fmt.Println(q.Symbol) }
for _, n := range resp.NewsRows() { fmt.Println(n["title"]) }
for _, l := range resp.Lists() { fmt.Println(l["slug"]) }
for _, r := range resp.Research() { fmt.Println(r["title"]) }

all := resp.All() // {"quotes": ..., "news": ..., "lists": ..., "researchReports": ...}
```

### Reliability and Observability

The client exposes four `Client` fields for hardening production traffic:

```go
client := yfinance.NewClient(nil)
client.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
client.Retries = 3
client.RetryBackoff = 250 * time.Millisecond
client.Limiter = yfinance.NewRateLimiter(5, 2) // 5 req/s, burst 2
```

| Field | Behavior |
| --- | --- |
| `Logger` | When non-nil, every request and response is emitted at `slog.LevelDebug` with `method`, `url`, `status`. Mirrors yfinance's `enable_debug_mode()`. |
| `Retries` | Bounds retries for transient failures (`5xx`, `ErrRateLimited`, temporary network errors). Backoff doubles on each attempt. |
| `RetryBackoff` | Base wait between retries. Defaults to 250 ms when zero. |
| `Limiter` | Any value satisfying the `Limiter` interface (`Wait(ctx) error`). Use `NewRateLimiter(rps, burst)` for an in-package token bucket, or pass `golang.org/x/time/rate.Limiter`. |

`*RateLimiter` is a token bucket: tokens accrue at `rps` per second up to a `burst` cap. A zero or negative rate makes `Wait` a no-op.

### Authentication (Cookies + Crumb)

Some Yahoo endpoints — most notably `quoteSummary`, `v7/finance/quote`, and the fundamentals timeseries — require a `crumb` token paired with a session cookie. `goyfinace` handles this automatically when the client targets the real Yahoo hosts:

```go
client := yfinance.NewClient(nil)

// Optional: pre-fetch the crumb up front. Otherwise the first call to a
// crumb-protected endpoint will trigger Authenticate transparently.
if err := client.Authenticate(ctx); err != nil {
    return err
}

fmt.Println(client.Crumb)
```

When you pass your own `*http.Client`, ensure it has a cookie jar — otherwise the package installs `cookiejar.New(nil)` for you. Tests that override `Query1URL`/`Query2URL` to point at `httptest` skip the auto-authentication step, so existing test suites do not need to mock the crumb endpoint.

### Quarterly / Annual Shares

```go
shares, err := client.Ticker("AAPL").Shares(ctx, "quarterly")
```

`Shares` wraps `FundamentalsTimeseries` for the `ShareIssued` and `OrdinarySharesNumber` types. Pass `"quarterly"`, `"annual"` (default when empty), or `"trailing"`.

### Adjusted History

`HistoryParams.AutoAdjust`, `BackAdjust`, and `Rounding` mirror yfinance's pricing adjustment flags. They are applied client-side to the candles returned by Yahoo's chart endpoint, so no extra request is needed.

```go
history, err := client.Ticker("AAPL").History(ctx, yfinance.HistoryParams{
    Period:     yfinance.Period1Y,
    Interval:   yfinance.Interval1D,
    AutoAdjust: true,
    Rounding:   true,
})
```

## Advanced Search

`SearchWithOptions` exposes the full Yahoo search query surface. Pointer-bool fields make it possible to differentiate "leave at server default" from an explicit `false`:

```go
yes := true
resp, err := client.SearchWithOptions(ctx, "apple", yfinance.SearchOptions{
    QuotesCount:                10,
    NewsCount:                  5,
    ListsCount:                 3,
    EnableFuzzyQuery:           &yes,
    EnableEnhancedTrivialQuery: &yes,
    EnablePrivateCompany:       &yes,
    RecommendCount:             5,
    Region:                     "GB",
    Lang:                       "en-GB",
})
```

`Search(query, quotesCount, newsCount)` remains the simple two-argument form.

## Response Cache

`Client.Cache` accepts any value implementing the `Cache` interface (`Get`, `Set`). When set, GET-JSON requests check the cache before issuing an HTTP request and store successful responses keyed by full URL. `Client.CacheTTL` controls expiry; zero means no expiry.

```go
client := yfinance.NewClient(nil)
client.Cache = yfinance.NewMemoryCache()
client.CacheTTL = 5 * time.Minute
```

`*MemoryCache` is a process-local map. Plug in a Redis-backed or filesystem-backed implementation by satisfying the `Cache` interface.

## WebSocket Auto-Reconnect

```go
ws := client.WebSocket("")
ws.AutoReconnect = true
ws.ReconnectBackoff = 500 * time.Millisecond
ws.MaxReconnectAttempts = 0 // 0 = unlimited
ws.OnReconnect = func(attempt int, err error) { /* observability */ }

if err := ws.Subscribe(ctx, "AAPL", "MSFT"); err != nil {
    return err
}

err := ws.Listen(ctx, func(msg yfinance.StreamMessage) {
    fmt.Println(msg.ID, msg.Price)
})
```

When `AutoReconnect` is true, transport errors trigger an exponential-backoff reconnect (capped at 30 s) and a re-`Subscribe` of every tracked symbol before resuming. `OnReconnect` is fired before each attempt with the attempt number and triggering error.

## Versioning

Module path:

```text
github.com/Nightsuki/goyfinace
```

Install a specific version:

```sh
go get github.com/Nightsuki/goyfinace@v0.5.0
```

Go package documentation is available after pkg.go.dev indexes the module:

```text
https://pkg.go.dev/github.com/Nightsuki/goyfinace
```
