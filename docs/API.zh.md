# goyfinace API 文档

English version: [API.md](API.md)

本文档面向 Go 调用方，说明 `github.com/Nightsuki/goyfinace` 当前公开 API、参数语义、返回结构和错误处理方式。包名是 `yfinance`，建议导入时显式使用别名：

```go
import yfinance "github.com/Nightsuki/goyfinace"
```

## 设计边界

`goyfinace` 是 Yahoo Finance 公共接口的 Go 客户端，实现了 Python `yfinance` 的常用使用路径：

- 历史 K 线：`History`
- 多标的并发下载：`Download`
- 常用 quote 信息：`Info`
- 轻量价格信息：`FastInfo`
- 原始 quote summary 模块：`QuoteSummary`
- 期权链：`Options`
- 搜索：`Search`
- 财务报表、持仓、分析师建议：`Financials`、`Holders`、`Recommendations`

Yahoo Finance 没有为这些端点提供正式稳定的公共 API，因此生产环境中应处理限流、字段缺失、响应结构变化和网络错误。

## 安装

```sh
go get github.com/Nightsuki/goyfinace
```

## 快速开始

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

创建 Yahoo Finance 客户端。传入 `nil` 时会使用默认 `http.Client`，超时时间为 15 秒。

### Client 字段

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

字段说明：

| 字段 | 说明 |
| --- | --- |
| `HTTPClient` | 所有请求使用的 HTTP 客户端。可用于设置超时、代理、Transport、重试包装等。 |
| `Query1URL` | Yahoo `query1.finance.yahoo.com` 的基础地址。测试或私有代理可覆盖。 |
| `Query2URL` | Yahoo `query2.finance.yahoo.com` 的基础地址。测试或私有代理可覆盖。 |
| `RootURL` | Yahoo Finance 前端 JSON 端点的基础地址，例如新闻接口。 |
| `ISINURL` | 外部 best-effort ISIN suggestion 端点。Yahoo 没有稳定的 ISIN lookup，因此该地址与 Yahoo 端点分离。 |
| `UserAgent` | 请求头中的 User-Agent。为空时使用 `DefaultUserAgents` 中的默认浏览器 UA。 |

`Client` 可以复用。建议在应用初始化时创建一个客户端，在业务代码中共享。

### Ticker

```go
func (c *Client) Ticker(symbol string) *Ticker
```

创建一个绑定股票代码的 helper。`Ticker` 会把 symbol 去空格并转大写：

```go
ticker := client.Ticker("msft")
info, err := ticker.Info(ctx)
history, err := ticker.History(ctx, yfinance.HistoryParams{Period: yfinance.Period5D})
```

`Client` 方法适合一次性调用，`Ticker` 方法适合同一个 symbol 的多次调用。两者返回的数据一致。

## 历史行情：History

### 方法

```go
func (c *Client) History(ctx context.Context, symbol string, params HistoryParams) (*HistoryResult, error)
func (t *Ticker) History(ctx context.Context, params HistoryParams) (*HistoryResult, error)
```

从 Yahoo chart 端点下载 OHLCV K 线。

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

参数说明：

| 字段 | 说明 |
| --- | --- |
| `Period` | Yahoo range 参数。只有 `Start` 和 `End` 都为空时使用。为空默认 `Period1Mo`。 |
| `Interval` | K 线粒度。为空默认 `Interval1D`。 |
| `Start` | 显式时间范围开始，包含该时间。设置 `Start` 或 `End` 后会使用 `period1/period2`。 |
| `End` | 显式时间范围结束，不包含该时间。`Start` 非空且 `End` 为空时使用当前时间。 |
| `PrePost` | 是否请求盘前盘后数据。仅在 Yahoo 对该标的和周期支持时有效。 |
| `Events` | 请求的事件类型。为空时请求 `div`、`splits`、`capitalGains`。 |
| `AutoAdjust` | 使用 `AdjClose/Close` 比例对 OHLC 进行除权/除息调整，并反向调整 Volume。对应 yfinance 的 `auto_adjust=True`。 |
| `BackAdjust` | 反向调整历史 K 线，使最近一根 Close 等于 AdjClose，保持最新原始价格不变。对应 yfinance 的 `back_adjust=True`。 |
| `Rounding` | 按照图表元数据中的 `PriceHint` 对 OHLC 进行四舍五入。对应 yfinance 的 `rounding=True`。 |
| `Repair` | 通过对比 Close 与邻域中位数，自动修复 100 倍/百分之一倍的脏数据（多由货币单位变化导致）。对应 yfinance 的 `repair=True`。 |
| `Threads` | 限制 `Download` 的并发数；零或负值表示无上限。`History` 不使用此字段。对应 yfinance 的 `download(threads=N)`。 |
| `NoEvents` | 显式禁用 dividend/split/capital-gain 事件。默认情况下 Yahoo 会附带事件。 |
| `OnProgress` | 可选回调 `func(symbol, idx, total int, err error)`，由 `Download` 在每个 symbol 处理完后调用，对应 yfinance 的 tqdm 进度回调。 |
| `DropNaN` | 丢弃所有 OHLC 全为 NaN 的 K 线。对应 yfinance 的 `dropna=True`。 |

### Period 常量

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

### Interval 常量

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

Yahoo 对分钟级数据通常有历史窗口限制。若请求范围过大，Yahoo 可能返回错误或空结果。

### HistoryResult

```go
type HistoryResult struct {
	Symbol   string
	Meta     ChartMeta
	Candles  []Candle
	YahooErr *YahooError
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `Symbol` | 标准化后的 symbol。 |
| `Meta` | Yahoo chart metadata 中常用字段。 |
| `Candles` | K 线数组。 |
| `YahooErr` | Yahoo JSON 返回结构化错误时填充。该错误也会作为 `error` 返回。 |

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

字段说明：

| 字段 | 说明 |
| --- | --- |
| `Time` | UTC 时间戳。 |
| `Open` / `High` / `Low` / `Close` | 原始 OHLC 价格。Yahoo 返回 `null` 时为 `math.NaN()`。 |
| `AdjClose` | 复权收盘价。Yahoo 未返回时回退为 `Close`。 |
| `Volume` | 成交量。缺失时为 0。 |
| `Dividends` | 当前时间戳上的分红金额，没有则为 0。 |
| `Split` | 当前时间戳上的拆股比例，例如 4-for-1 表示为 `4`。没有则为 0。 |

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

`Raw` 保存 Yahoo 返回的完整 metadata。若需要尚未提升为强类型字段的 Yahoo 原始字段，可从 `Raw` 中读取。

### 示例：按 period 下载

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

### 示例：按日期范围下载

```go
history, err := client.History(ctx, "MSFT", yfinance.HistoryParams{
	Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	End:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	Interval: yfinance.Interval1D,
})
```

`End` 是排他边界，与 Python `yfinance` 和 Yahoo chart 行为一致。

## 多标的下载：Download

```go
func (c *Client) Download(ctx context.Context, symbols []string, params HistoryParams) []DownloadResult
```

并发下载多个 symbol 的历史行情。返回数组顺序与输入 `symbols` 一致。

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

`Download` 不会因为单个 symbol 失败而提前终止。调用方需要逐个检查 `DownloadResult.Err`。

```go
type DownloadResult struct {
	Symbol string
	Result *HistoryResult
	Err    error
}
```

## DataFrame 适配器：Gota

根包默认返回 Go 原生值，不直接暴露 DataFrame 类型。对于从 pandas 风格 `yfinance` 迁移过来的用户，`goyfinace` 提供可选 Gota 适配器：

```go
import yfgota "github.com/Nightsuki/goyfinace/adapter/gota"
```

适配器位于独立 import path，因此只使用核心 HTTP 客户端的应用无需直接使用 DataFrame API。

### 历史行情 DataFrame

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

`yfgota.History` 和 `yfgota.Candles` 会生成接近 Python `yfinance` 使用习惯的列名：

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

`Time` 是 UTC RFC3339 字符串。`Timestamp` 是 Unix 秒数，便于数值排序或过滤。

### 期权 DataFrame

```go
chain, err := client.Options(ctx, "AAPL", time.Time{})
if err != nil {
	return err
}

calls, puts := yfgota.Options(chain)
fmt.Println(calls.Nrow(), puts.Nrow())
```

也可以单独转换一侧：

```go
calls := yfgota.OptionContracts(chain.Calls)
```

期权 DataFrame 使用 Yahoo option-chain 字段名，例如 `contractSymbol`、`lastTradeDate`、`strike`、`lastPrice`、`bid`、`ask`、`volume`、`openInterest`、`impliedVolatility`。

### Search 与 QuoteSummary DataFrame

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

`QuoteSummary` 返回长表结构，包含 `Module`、`Key`、`Value` 三列。Yahoo `{raw, fmt}` 值会优先展开为 `raw`，嵌套对象会转成 JSON 字符串，以便放入 DataFrame 的标量单元格。

### 通用 map 和 records helper

```go
kv := yfgota.KeyValues(info)
records := yfgota.Records([]map[string]any{
	{"symbol": "AAPL", "regularMarketPrice": 201.5},
})
```

这些 helper 可用于财务模块、holders、recommendations 或调用方自行整理后的记录。

## QuoteSummary

```go
func (c *Client) QuoteSummary(ctx context.Context, symbol string, modules ...string) (map[string]any, error)
func (t *Ticker) QuoteSummary(ctx context.Context, modules ...string) (map[string]any, error)
```

直接访问 Yahoo quoteSummary 端点。适合高级调用方读取未封装的模块。

```go
summary, err := client.QuoteSummary(ctx, "AAPL", "price", "summaryDetail")
if err != nil {
	return err
}
price := summary["price"].(map[string]any)
fmt.Println(price["symbol"])
```

当 `modules` 为空时，会请求 `Info` 使用的默认模块：

- `price`
- `summaryDetail`
- `financialData`
- `quoteType`
- `defaultKeyStatistics`
- `assetProfile`

返回值是 Yahoo 原始模块 map。由于 Yahoo 字段经常变化，建议业务层对类型断言做保护。

## Info

```go
func (c *Client) Info(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Info(ctx context.Context) (map[string]any, error)
```

请求常用 quoteSummary 模块，并把模块内字段扁平化为单层 `map[string]any`。Yahoo 的 `{raw, fmt}` 数值对象会优先展开为 `raw`，没有 `raw` 时使用 `fmt`。

```go
info, err := client.Info(ctx, "AAPL")
if err != nil {
	return err
}

fmt.Println(info["symbol"])
fmt.Println(info["regularMarketPrice"])
fmt.Println(info["marketCap"])
```

适用场景：

- 快速读取公司名称、市值、币种、交易所等常用字段。
- 需要接近 Python `Ticker.info` 的 map 风格。

不适用场景：

- 需要稳定 schema 的核心业务。此时建议使用 `QuoteSummary` 指定模块并在业务层定义自己的结构。

## FastInfo

```go
func (c *Client) FastInfo(ctx context.Context, symbol string) (*FastInfo, error)
func (t *Ticker) FastInfo(ctx context.Context) (*FastInfo, error)
```

通过 chart metadata 获取轻量价格信息。当前实现会请求 `5d/1d` chart metadata。

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

## 搜索：Search

```go
func (c *Client) Search(ctx context.Context, query string, quotesCount, newsCount int) (*SearchResponse, error)
```

搜索股票、ETF、指数等 Yahoo Finance symbol。

```go
resp, err := client.Search(ctx, "apple", 10, 0)
if err != nil {
	return err
}
for _, quote := range resp.Quotes {
	fmt.Println(quote.Symbol, quote.ShortName, quote.QuoteType)
}
```

参数说明：

| 参数 | 说明 |
| --- | --- |
| `query` | 搜索关键词，不能为空。 |
| `quotesCount` | 返回 quote 数量。小于等于 0 时默认 10。 |
| `newsCount` | 返回新闻数量。小于 0 时按 0 处理。 |

返回结构：

```go
type SearchResponse struct {
	Quotes []SearchResult
	News   []any
	Raw    map[string]any
}
```

`Raw` 保存完整 Yahoo 响应，便于读取未强类型化字段。

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

## 期权链：Options

```go
func (c *Client) Options(ctx context.Context, symbol string, expiration time.Time) (*OptionChain, error)
func (t *Ticker) Options(ctx context.Context, expiration time.Time) (*OptionChain, error)
```

获取期权链。`expiration` 传零值 `time.Time{}` 时，请求 Yahoo 默认到期日。

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

如果要请求指定到期日，先读取 `Expirations`，再用其中某个日期再次请求：

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

返回结构：

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

`Underlying` 是 Yahoo 返回的底层标的 quote 原始对象。

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

`ImpliedVolatility` 是小数形式，例如 0.25 表示 25%。

## 财务、持仓、推荐

这些方法都是 `QuoteSummary` 的便利封装，返回 Yahoo 原始模块 map。

### Financials

```go
func (c *Client) Financials(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Financials(ctx context.Context) (map[string]any, error)
```

请求模块：

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

请求模块：

- `institutionOwnership`
- `fundOwnership`
- `majorHoldersBreakdown`
- `insiderHolders`

### Recommendations

```go
func (c *Client) Recommendations(ctx context.Context, symbol string) (map[string]any, error)
func (t *Ticker) Recommendations(ctx context.Context) (map[string]any, error)
```

请求模块：

- `recommendationTrend`
- `upgradeDowngradeHistory`

## 错误处理

包内定义了两个哨兵错误：

```go
var ErrNoResult = errors.New("yfinance: no result")
var ErrRateLimited = errors.New("yfinance: rate limited")
```

使用方式：

```go
history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{})
if err != nil {
	switch {
	case errors.Is(err, yfinance.ErrRateLimited):
		// 限流：退避重试或降低并发
	case errors.Is(err, yfinance.ErrNoResult):
		// symbol 不存在、无权限或 Yahoo 无数据
	default:
		var yahooErr *yfinance.YahooError
		if errors.As(err, &yahooErr) {
			fmt.Println(yahooErr.Code, yahooErr.Description)
		}
		return err
	}
}
```

`YahooError` 保留 Yahoo JSON 中的错误对象：

```go
type YahooError struct {
	Code        string
	Description string
}
```

另外，HTTP 非 2xx 响应会返回包含状态码和响应片段的普通 `error`。

## 其他 yfinance helper

### 多标的 Tickers

```go
tickers := client.Tickers("AAPL MSFT")
quotes, err := tickers.Quotes(ctx)
downloads := tickers.Download(ctx, yfinance.HistoryParams{Period: yfinance.Period1Mo})
```

`Tickers` 保存标准化后的 symbols，并委托 `Quote` 和 `Download` 执行。

### 筛选器 query builder

```go
query := yfinance.EquityQuery(
	yfinance.GTE("intradaymarketcap", 1_000_000_000),
	yfinance.Between("eodvolume", 1_000_000, 10_000_000),
)
result, err := client.Screen(ctx, yfinance.ScreenRequest{Query: query, Size: 50})
```

`EquityQuery`、`FundQuery`、`ETFQuery` 会生成 Yahoo screener query map。底层构造器包括 `Eq`、`GT`、`GTE`、`LT`、`LTE`、`Between`、`And`、`Or`、`Screener`。

### estimate、holder 和 insider helper

以下 helper 是 quoteSummary 的轻量封装，返回 Yahoo 原始模块 payload：

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

### Market 与 Calendars

```go
summary, err := client.Market("us").Summary(ctx)
status, err := client.Market("us").Status(ctx)
earnings, err := client.Calendars().Earnings(ctx, 25)
```

### WebSocket 流式行情

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

`NewAsyncWebSocket` 和 `Client.AsyncWebSocket` 返回同一个由 context 驱动的 Go client。Yahoo 推送帧会解码为 `StreamMessage`，原始值保留在 `Raw` 中。

## Context 与超时

所有网络方法都接收 `context.Context`，推荐在业务层设置超时：

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

history, err := client.History(ctx, "AAPL", yfinance.HistoryParams{})
```

也可以传入自定义 `http.Client`：

```go
client := yfinance.NewClient(&http.Client{
	Timeout: 5 * time.Second,
})
```

## 测试与代理

`Client.Query1URL` 和 `Client.Query2URL` 可用于测试或接入内部代理：

```go
client := yfinance.NewClient(httpClient)
client.Query1URL = "http://127.0.0.1:8080"
client.Query2URL = "http://127.0.0.1:8080"
```

仓库单元测试使用 `httptest` 模拟 Yahoo 响应，不依赖真实网络。

## 与 Python yfinance 的差异

主要差异：

- Go 版本不返回 DataFrame，而是返回结构体、切片和 map。
- 可选 DataFrame 支持位于 `github.com/Nightsuki/goyfinace/adapter/gota`，但它不是完整 pandas 克隆。
- `Info` 返回扁平化 map，但不保证字段集合固定。
- financial、screener、calendar、domain、analysis 等接口保留 Yahoo 原始模块/端点结构，调用方可以按需要建模。
- Python 专属 scraping、`repair` 启发式修复、完整 pandas index/MultiIndex 行为以及 Python `asyncio` API 不在根包中复刻。

## yfinance 兼容功能面

除核心 API 外，本包为主要 yfinance 功能家族提供 Go 原生入口：

- 公司行为：`Actions`、`Dividends`、`Splits`、`CapitalGains`。
- Quote 和 quote-summary 扩展：`Quote`、`Calendar`、`SECFilings`、`Sustainability`、`Valuation`。
- 分析师数据：`Analysis`、`AnalystPriceTargets`、`UpgradesDowngrades`、`Recommendations`。
- 基金和持仓：`FundProfile`、`FundsData`、`Holders`。
- 财务报表和股本：`FundamentalsTimeseries`、`IncomeStatement`、`BalanceSheet`、`CashFlow`、`SharesFull`。
- 估值、持仓和 insider：`EarningsEstimate`、`RevenueEstimate`、`EarningsHistory`、`EPSRevisions`、`EPSTrend`、`GrowthEstimates`、`RecommendationsSummary`、`MajorHolders`、`InstitutionalHolders`、`MutualFundHolders`、`InsiderPurchases`、`InsiderTransactions`、`InsiderRosterHolders`。
- 发现和筛选器：`Lookup`、`LookupISIN`、`Search`、`Screen`、`PredefinedScreen`、`EquityQuery`、`FundQuery`、`ETFQuery`。
- 市场和 domain 数据：`Market`、`MarketSummary`、`MarketStatus`、`Sector`、`Industry`、`SectorOf`、`IndustryOf`。
- 日历和新闻：`CalendarVisualization`、`EarningsDates`、`News`。
- 多标的和流式行情：`Tickers`、`WebSocket`、`AsyncWebSocket`。

当 Yahoo schema 较宽或不稳定时，这些方法会返回原始 `map[string]any` 或简单 records。需要表格表达时，可使用 `adapter/gota` 的 `QuoteSummary`、`KeyValues`、`Records`、`Actions`、`Timeseries` 等 helper。

建议迁移方式：

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

## 类型化领域访问器

除原始的 `Sector`/`Industry`/`FundProfile` map 接口外，本包还提供与 yfinance 中 `Sector`、`Industry`、`FundsData` 类属性一一对应的类型化封装。

### SectorOf / IndustryOf

```go
sector, err := client.SectorOf(ctx, "technology")
if err != nil {
    return err
}

fmt.Println(sector.Name)
fmt.Println(sector.Overview())
for _, row := range sector.TopCompanies() { /* ... */ }
sector.TopETFs()
sector.TopMutualFunds()
sector.Industries()
sector.TopGrowthCompanies()
sector.TopPerformingCompanies()

industry, err := client.IndustryOf(ctx, "software-application")
if err != nil {
    return err
}
industry.Overview()
industry.TopPerformingCompanies()
industry.TopGrowthCompanies()
industry.KeyCompanyKeys()
industry.KeyCompanyGroups()
```

`SectorData.Raw` 和 `IndustryData.Raw` 保留原始响应，便于访问类型方法未提升的字段。

### FundsData

```go
funds, err := client.Ticker("VGT").FundsData(ctx)
if err != nil {
    return err
}

funds.Description()
funds.FundOverview()    // family / category / legalType
funds.FundOperations()  // 费用率
funds.AssetClasses()    // cashPosition、stockPosition 等
funds.TopHoldings()
funds.EquityHoldings()
funds.BondHoldings()
funds.BondRatings()
funds.SectorWeightings()
```

### Search 各分区访问器

`SearchResponse` 暴露 Yahoo 搜索结果的全部主要分区：

```go
resp, err := client.Search(ctx, "apple", 10, 5)
if err != nil {
    return err
}

resp.Quotes
resp.NewsRows()
resp.Lists()
resp.Research()
resp.All() // 同时聚合上述所有分区
```

### 可靠性与可观测性

客户端提供四个字段用于在生产环境中加固请求行为：

```go
client := yfinance.NewClient(nil)
client.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
client.Retries = 3
client.RetryBackoff = 250 * time.Millisecond
client.Limiter = yfinance.NewRateLimiter(5, 2) // 每秒 5 次，突发 2 次
```

| 字段 | 行为 |
| --- | --- |
| `Logger` | 非空时，每个请求和响应会以 `slog.LevelDebug` 输出 `method`/`url`/`status`。对应 yfinance 的 `enable_debug_mode()`。 |
| `Retries` | 限定重试次数；只对瞬时错误（5xx、`ErrRateLimited`、临时网络错误）重试，每次退避翻倍。 |
| `RetryBackoff` | 重试基础等待时间，零值默认为 250ms。 |
| `Limiter` | 任意实现 `Limiter`（`Wait(ctx) error`）的值。使用 `NewRateLimiter(rps, burst)` 获得包内令牌桶，也可使用 `golang.org/x/time/rate.Limiter`。 |

`*RateLimiter` 是一个令牌桶，按 `rps` 速率累计令牌，最多累计到 `burst`。当速率为零或负数时 `Wait` 直接返回。

### 认证（Cookie + Crumb）

部分 Yahoo 接口（`quoteSummary`、`v7/finance/quote`、fundamentals timeseries 等）需要带上由 cookie 派生出的 `crumb` 令牌。当客户端访问真实 Yahoo 域名时，本包会自动处理：

```go
client := yfinance.NewClient(nil)

// 可选：提前完成认证；否则首次访问受保护接口时会自动触发
if err := client.Authenticate(ctx); err != nil {
    return err
}

fmt.Println(client.Crumb)
```

如果你传入自定义 `*http.Client`，会自动为其挂载 `cookiejar.New(nil)`。测试通过 `Query1URL`/`Query2URL` 重定向到 `httptest` 时会跳过自动认证，已有测试无需新增 mock。

### 季度 / 年度股本

```go
shares, err := client.Ticker("AAPL").Shares(ctx, "quarterly")
```

`Shares` 在 `FundamentalsTimeseries` 之上请求 `ShareIssued` 和 `OrdinarySharesNumber` 两个序列。`freq` 取值为 `"quarterly"`、`"annual"`（默认）或 `"trailing"`。

### 价格调整与四舍五入

`HistoryParams.AutoAdjust`、`BackAdjust`、`Rounding` 与 yfinance 中的同名参数语义一致。这些调整在 Yahoo 图表数据返回后由客户端本地完成，不会触发额外请求。

```go
history, err := client.Ticker("AAPL").History(ctx, yfinance.HistoryParams{
    Period:     yfinance.Period1Y,
    Interval:   yfinance.Interval1D,
    AutoAdjust: true,
    Rounding:   true,
})
```

## 高级搜索

`SearchWithOptions` 暴露 Yahoo 搜索接口的完整参数。布尔字段使用 `*bool`，可以区分"使用服务端默认"与"显式 false"：

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

`Search(query, quotesCount, newsCount)` 仍然保留为简化入口。

## 响应缓存

`Client.Cache` 接受任意实现 `Cache` 接口（`Get`/`Set`）的值。设置后，GET-JSON 请求会在发起 HTTP 调用前查询缓存，并在成功后按完整 URL 缓存响应。`Client.CacheTTL` 控制过期时间，零值表示永不过期。

```go
client := yfinance.NewClient(nil)
client.Cache = yfinance.NewMemoryCache()
client.CacheTTL = 5 * time.Minute
```

`*MemoryCache` 是进程内 map 实现。要接入 Redis 或文件缓存，自行实现 `Cache` 接口即可。

## WebSocket 自动重连

```go
ws := client.WebSocket("")
ws.AutoReconnect = true
ws.ReconnectBackoff = 500 * time.Millisecond
ws.MaxReconnectAttempts = 0 // 0 表示无限重连
ws.OnReconnect = func(attempt int, err error) { /* 监控 */ }

if err := ws.Subscribe(ctx, "AAPL", "MSFT"); err != nil {
    return err
}

err := ws.Listen(ctx, func(msg yfinance.StreamMessage) {
    fmt.Println(msg.ID, msg.Price)
})
```

`AutoReconnect=true` 时，传输错误会触发指数退避（封顶 30 秒）的重连流程，并在重连成功后自动 `Subscribe` 所有已追踪的 symbol。`OnReconnect` 在每次尝试前回调，参数包含当前尝试次数和触发错误。

## 发布与版本

当前模块路径：

```text
github.com/Nightsuki/goyfinace
```

安装指定版本：

```sh
go get github.com/Nightsuki/goyfinace@v0.5.0
```

Go 文档发布后可在 pkg.go.dev 查看：

```text
https://pkg.go.dev/github.com/Nightsuki/goyfinace
```
