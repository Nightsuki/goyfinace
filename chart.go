package yfinance

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"
)

const (
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
)

// HistoryParams controls chart downloads.
type HistoryParams struct {
	// Period is a Yahoo range value such as Period1D, Period1Mo, Period1Y, or
	// PeriodMax. It is used only when Start and End are both zero. If empty, the
	// client defaults to Period1Mo.
	Period string
	// Interval is the candle size, for example Interval1D or Interval1H. If
	// empty, the client defaults to Interval1D.
	Interval string
	// Start is the inclusive beginning of an explicit date range. Set Start or
	// End to use period1/period2 instead of range.
	Start time.Time
	// End is the exclusive end of an explicit date range. If Start is set and
	// End is zero, the current time is used.
	End time.Time
	// PrePost includes pre-market and post-market rows when Yahoo supports them.
	PrePost bool
	// Events controls event data included by Yahoo. If empty, dividends, splits,
	// and capital gains are requested.
	Events []string
}

// Candle is one OHLCV row from Yahoo chart data.
type Candle struct {
	// Time is the candle timestamp in UTC.
	Time time.Time
	// Open, High, Low, and Close are the raw OHLC prices. Yahoo null values are
	// represented as math.NaN for floating point fields.
	Open  float64
	High  float64
	Low   float64
	Close float64
	// AdjClose is Yahoo's adjusted close value. If Yahoo omits adjusted close,
	// the client falls back to Close for that row.
	AdjClose float64
	// Volume is the reported trading volume. Missing volume is zero.
	Volume int64
	// Dividends contains the dividend amount for this timestamp when Yahoo
	// returns a dividend event.
	Dividends float64
	// Split is numerator/denominator for a split event at this timestamp. For
	// example, a 4-for-1 split is represented as 4.
	Split float64
}

// ChartMeta contains selected metadata returned by Yahoo chart.
type ChartMeta struct {
	Currency          string
	Symbol            string
	ExchangeName      string
	FullExchangeName  string
	InstrumentType    string
	FirstTradeDate    time.Time
	RegularMarketTime time.Time
	// GMTOffset is Yahoo's exchange offset in seconds.
	GMTOffset int
	Timezone  string
	// ExchangeTimezoneName is the IANA timezone name when Yahoo provides one.
	ExchangeTimezoneName string
	RegularMarketPrice   float64
	ChartPreviousClose   float64
	PriceHint            int
	// Raw contains the full chart meta object for callers that need fields not
	// promoted by this struct.
	Raw map[string]any
}

// HistoryResult contains candles and response metadata for one symbol.
type HistoryResult struct {
	Symbol  string
	Meta    ChartMeta
	Candles []Candle
	// YahooErr is set when Yahoo returned a structured chart error. The same
	// error is also returned from History.
	YahooErr *YahooError
}

// History downloads historical OHLCV candles for this ticker.
func (t *Ticker) History(ctx context.Context, params HistoryParams) (*HistoryResult, error) {
	return t.c().History(ctx, t.Symbol, params)
}

// History downloads historical OHLCV candles for a symbol.
func (c *Client) History(ctx context.Context, symbol string, params HistoryParams) (*HistoryResult, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("yfinance: empty symbol")
	}
	if params.Interval == "" {
		params.Interval = Interval1D
	}
	q := url.Values{}
	q.Set("interval", params.Interval)
	q.Set("includePrePost", strconv.FormatBool(params.PrePost))
	q.Set("events", eventsParam(params.Events))
	if !params.Start.IsZero() || !params.End.IsZero() {
		start := params.Start
		if start.IsZero() {
			start = time.Unix(0, 0)
		}
		end := params.End
		if end.IsZero() {
			end = time.Now()
		}
		q.Set("period1", strconv.FormatInt(start.Unix(), 10))
		q.Set("period2", strconv.FormatInt(end.Unix(), 10))
	} else {
		if params.Period == "" {
			params.Period = Period1Mo
		}
		q.Set("range", params.Period)
	}

	var resp chartResponse
	err := c.getJSON(ctx, c.cloneWithDefaults().Query1URL, "/v8/finance/chart/"+url.PathEscape(symbol), q, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Chart.Error != nil {
		return &HistoryResult{Symbol: symbol, YahooErr: resp.Chart.Error}, resp.Chart.Error
	}
	if len(resp.Chart.Result) == 0 {
		return nil, ErrNoResult
	}
	result := resp.Chart.Result[0]
	return decodeChartResult(symbol, result), nil
}

func eventsParam(events []string) string {
	if len(events) == 0 {
		return "div,splits,capitalGains"
	}
	out := events[0]
	for _, ev := range events[1:] {
		out += "," + ev
	}
	return out
}

func decodeChartResult(symbol string, result chartResult) *HistoryResult {
	quotes := chartQuote{}
	if len(result.Indicators.Quote) > 0 {
		quotes = result.Indicators.Quote[0]
	}
	var adj []jsonNumber
	if len(result.Indicators.AdjClose) > 0 {
		adj = result.Indicators.AdjClose[0].AdjClose
	}
	dividends := map[int64]float64{}
	for key, ev := range result.Events.Dividends {
		ts, err := strconv.ParseInt(key, 10, 64)
		if err == nil {
			dividends[ts] = ev.Amount.Float64()
		}
	}
	splits := map[int64]float64{}
	for key, ev := range result.Events.Splits {
		ts, err := strconv.ParseInt(key, 10, 64)
		if err == nil && ev.Denominator.Float64() != 0 {
			splits[ts] = ev.Numerator.Float64() / ev.Denominator.Float64()
		}
	}

	candles := make([]Candle, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		closeValue := numberAt(quotes.Close, i)
		adjClose := numberAt(adj, i)
		if math.IsNaN(adjClose) {
			adjClose = closeValue
		}
		candles = append(candles, Candle{
			Time:      time.Unix(ts, 0).UTC(),
			Open:      numberAt(quotes.Open, i),
			High:      numberAt(quotes.High, i),
			Low:       numberAt(quotes.Low, i),
			Close:     closeValue,
			AdjClose:  adjClose,
			Volume:    int64At(quotes.Volume, i),
			Dividends: dividends[ts],
			Split:     splits[ts],
		})
	}
	return &HistoryResult{
		Symbol:  symbol,
		Meta:    decodeMeta(result.Meta),
		Candles: candles,
	}
}

func decodeMeta(meta map[string]any) ChartMeta {
	out := ChartMeta{Raw: meta}
	out.Currency = stringValue(meta["currency"])
	out.Symbol = stringValue(meta["symbol"])
	out.ExchangeName = stringValue(meta["exchangeName"])
	out.FullExchangeName = stringValue(meta["fullExchangeName"])
	out.InstrumentType = stringValue(meta["instrumentType"])
	out.GMTOffset = int(numberValue(meta["gmtoffset"]))
	out.Timezone = stringValue(meta["timezone"])
	out.ExchangeTimezoneName = stringValue(meta["exchangeTimezoneName"])
	out.RegularMarketPrice = numberValue(meta["regularMarketPrice"])
	out.ChartPreviousClose = numberValue(meta["chartPreviousClose"])
	out.PriceHint = int(numberValue(meta["priceHint"]))
	if ts := int64(numberValue(meta["firstTradeDate"])); ts > 0 {
		out.FirstTradeDate = time.Unix(ts, 0).UTC()
	}
	if ts := int64(numberValue(meta["regularMarketTime"])); ts > 0 {
		out.RegularMarketTime = time.Unix(ts, 0).UTC()
	}
	return out
}

func numberAt(values []jsonNumber, i int) float64 {
	if i < 0 || i >= len(values) || !values[i].Valid {
		return math.NaN()
	}
	return values[i].Float64()
}

func int64At(values []jsonNumber, i int) int64 {
	if i < 0 || i >= len(values) || !values[i].Valid {
		return 0
	}
	return values[i].Int64()
}
