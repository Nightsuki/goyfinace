package yfinance

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
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
	// AutoAdjust applies Yahoo's adjusted-close ratio to Open, High, Low, and
	// Volume so OHLC are split- and dividend-adjusted. Mirrors yfinance's
	// auto_adjust=True behavior. Close becomes the adjusted close value.
	AutoAdjust bool
	// BackAdjust scales the entire history so the latest Close matches AdjClose,
	// preserving raw most-recent prices while back-adjusting older candles.
	// Mirrors yfinance's back_adjust=True behavior.
	BackAdjust bool
	// Rounding rounds OHLC to the chart meta's PriceHint number of decimals.
	// Mirrors yfinance's rounding=True behavior.
	Rounding bool
	// Repair runs a heuristic data-repair pass over the candles, fixing common
	// 100x or 0.01x price anomalies (often caused by Yahoo currency-unit
	// changes) and replacing zero-volume rows with NaN-like zero markers.
	// Mirrors yfinance's repair=True flag.
	Repair bool
	// Threads bounds concurrency in Download. Zero or negative means unlimited
	// (a goroutine per symbol). Mirrors yfinance's download(threads=N).
	Threads int
	// NoEvents suppresses dividend/split/capital-gain event requests. By
	// default Yahoo includes events when none are listed in Events.
	NoEvents bool
	// OnProgress, when non-nil, is invoked by Download once per symbol after
	// it completes (success or failure). idx is the zero-based index in the
	// input slice; total is len(symbols).
	OnProgress func(symbol string, idx, total int, err error)
	// DropNaN drops candles where every OHLC value is NaN, mirroring
	// yfinance's dropna=True behavior. Default is to keep all rows.
	DropNaN bool
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
	// CapitalGains contains the capital-gains distribution amount for this
	// timestamp when Yahoo returns a capital-gains event.
	CapitalGains float64
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
	if params.NoEvents {
		q.Set("events", "")
	} else {
		q.Set("events", eventsParam(params.Events))
	}
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
	out := decodeChartResult(symbol, result)
	applyHistoryAdjustments(out, params)
	return out, nil
}

func applyHistoryAdjustments(h *HistoryResult, params HistoryParams) {
	if h == nil || len(h.Candles) == 0 {
		return
	}
	if params.Repair {
		repairCandles(h.Candles)
	}
	if params.AutoAdjust {
		for i := range h.Candles {
			c := &h.Candles[i]
			if math.IsNaN(c.Close) || math.IsNaN(c.AdjClose) || c.Close == 0 {
				continue
			}
			ratio := c.AdjClose / c.Close
			c.Open *= ratio
			c.High *= ratio
			c.Low *= ratio
			c.Close = c.AdjClose
			if c.Volume > 0 && ratio != 0 {
				c.Volume = int64(float64(c.Volume) / ratio)
			}
		}
	} else if params.BackAdjust {
		last := h.Candles[len(h.Candles)-1]
		if !math.IsNaN(last.Close) && !math.IsNaN(last.AdjClose) && last.AdjClose != 0 {
			ratio := last.Close / last.AdjClose
			for i := range h.Candles {
				c := &h.Candles[i]
				if math.IsNaN(c.AdjClose) {
					continue
				}
				c.Open = c.Open * c.AdjClose / nz(c.Close, c.AdjClose)
				c.High = c.High * c.AdjClose / nz(c.Close, c.AdjClose)
				c.Low = c.Low * c.AdjClose / nz(c.Close, c.AdjClose)
				c.Close = c.AdjClose * ratio
				c.AdjClose = c.AdjClose * ratio
			}
		}
	}
	if params.Rounding && h.Meta.PriceHint > 0 {
		mult := math.Pow(10, float64(h.Meta.PriceHint))
		for i := range h.Candles {
			c := &h.Candles[i]
			c.Open = roundTo(c.Open, mult)
			c.High = roundTo(c.High, mult)
			c.Low = roundTo(c.Low, mult)
			c.Close = roundTo(c.Close, mult)
			c.AdjClose = roundTo(c.AdjClose, mult)
		}
	}
	if params.DropNaN {
		filtered := h.Candles[:0]
		for _, c := range h.Candles {
			if math.IsNaN(c.Open) && math.IsNaN(c.High) && math.IsNaN(c.Low) && math.IsNaN(c.Close) {
				continue
			}
			filtered = append(filtered, c)
		}
		h.Candles = filtered
	}
}

// repairCandles patches the most common Yahoo data anomalies: rows whose
// price has been multiplied or divided by 100 due to currency-unit changes
// (e.g. GBP -> GBp), and isolated zero/negative OHLC values surrounded by
// valid neighbors. The heuristic is conservative — it only acts when the
// neighborhood is unambiguous.
func repairCandles(candles []Candle) {
	if len(candles) < 3 {
		return
	}
	const window = 7
	closes := make([]float64, len(candles))
	for i := range candles {
		closes[i] = candles[i].Close
	}
	for i := range candles {
		c := &candles[i]
		if math.IsNaN(c.Close) || c.Close <= 0 {
			continue
		}
		ref := neighborMedian(closes, i, window)
		if ref <= 0 || math.IsNaN(ref) {
			continue
		}
		ratio := c.Close / ref
		switch {
		case ratio > 50 && ratio < 200:
			scale := 0.01
			c.Open *= scale
			c.High *= scale
			c.Low *= scale
			c.Close *= scale
			c.AdjClose *= scale
			closes[i] = c.Close
		case ratio > 0.005 && ratio < 0.02:
			scale := 100.0
			c.Open *= scale
			c.High *= scale
			c.Low *= scale
			c.Close *= scale
			c.AdjClose *= scale
			closes[i] = c.Close
		}
	}
	// Pass 2: NaN-out isolated non-positive OHLC values when neighbors are
	// healthy. yfinance's repair drops these because Yahoo occasionally
	// returns 0 for one or two corrupt OHLC fields on an otherwise valid bar.
	for i := range candles {
		c := &candles[i]
		ref := neighborMedian(closes, i, window)
		if math.IsNaN(ref) || ref <= 0 {
			continue
		}
		if !math.IsNaN(c.Open) && c.Open <= 0 {
			c.Open = math.NaN()
		}
		if !math.IsNaN(c.High) && c.High <= 0 {
			c.High = math.NaN()
		}
		if !math.IsNaN(c.Low) && c.Low <= 0 {
			c.Low = math.NaN()
		}
	}
	// Pass 3: detect adjusted-close ratio outliers — rows whose adj/close
	// ratio differs more than 5x from the median ratio of healthy neighbors.
	ratios := make([]float64, len(candles))
	for i, c := range candles {
		if math.IsNaN(c.Close) || c.Close <= 0 || math.IsNaN(c.AdjClose) {
			ratios[i] = math.NaN()
			continue
		}
		ratios[i] = c.AdjClose / c.Close
	}
	for i := range candles {
		c := &candles[i]
		if math.IsNaN(ratios[i]) {
			continue
		}
		ref := neighborMedian(ratios, i, window)
		if math.IsNaN(ref) || ref == 0 {
			continue
		}
		jump := ratios[i] / ref
		if jump > 5 || jump < 0.2 {
			c.AdjClose = c.Close * ref
		}
	}
}

func neighborMedian(values []float64, idx, window int) float64 {
	half := window / 2
	lo := idx - half
	if lo < 0 {
		lo = 0
	}
	hi := idx + half + 1
	if hi > len(values) {
		hi = len(values)
	}
	pool := make([]float64, 0, hi-lo)
	for j := lo; j < hi; j++ {
		if j == idx {
			continue
		}
		v := values[j]
		if math.IsNaN(v) || v <= 0 {
			continue
		}
		pool = append(pool, v)
	}
	if len(pool) == 0 {
		return math.NaN()
	}
	sort.Float64s(pool)
	mid := len(pool) / 2
	if len(pool)%2 == 1 {
		return pool[mid]
	}
	return (pool[mid-1] + pool[mid]) / 2
}

func nz(v, fallback float64) float64 {
	if math.IsNaN(v) || v == 0 {
		return fallback
	}
	return v
}

func roundTo(v, mult float64) float64 {
	if math.IsNaN(v) {
		return v
	}
	return math.Round(v*mult) / mult
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
	capitalGains := map[int64]float64{}
	for key, ev := range result.Events.CapitalGains {
		ts, err := strconv.ParseInt(key, 10, 64)
		if err == nil {
			capitalGains[ts] = ev.Amount.Float64()
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
			Time:         time.Unix(ts, 0).UTC(),
			Open:         numberAt(quotes.Open, i),
			High:         numberAt(quotes.High, i),
			Low:          numberAt(quotes.Low, i),
			Close:        closeValue,
			AdjClose:     adjClose,
			Volume:       int64At(quotes.Volume, i),
			Dividends:    dividends[ts],
			CapitalGains: capitalGains[ts],
			Split:        splits[ts],
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
