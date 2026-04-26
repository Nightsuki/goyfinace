package gota

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	yfinance "github.com/Nightsuki/goyfinace"
	"github.com/go-gota/gota/dataframe"
)

// History converts a HistoryResult into a DataFrame.
//
// The output uses column names familiar to yfinance users:
// Time, Timestamp, Open, High, Low, Close, Adj Close, Volume, Dividends, and
// Stock Splits.
func History(result *yfinance.HistoryResult) dataframe.DataFrame {
	if result == nil {
		return dataframe.LoadMaps(nil)
	}
	return Candles(result.Candles)
}

// Candles converts candle rows into a DataFrame.
func Candles(candles []yfinance.Candle) dataframe.DataFrame {
	rows := make([]map[string]any, 0, len(candles))
	for _, candle := range candles {
		rows = append(rows, map[string]any{
			"Time":         formatTime(candle.Time),
			"Timestamp":    candle.Time.Unix(),
			"Open":         finiteOrNaN(candle.Open),
			"High":         finiteOrNaN(candle.High),
			"Low":          finiteOrNaN(candle.Low),
			"Close":        finiteOrNaN(candle.Close),
			"Adj Close":    finiteOrNaN(candle.AdjClose),
			"Volume":       candle.Volume,
			"Dividends":    candle.Dividends,
			"Stock Splits": candle.Split,
		})
	}
	return dataframe.LoadMaps(rows)
}

// Options converts an option chain into call and put DataFrames.
func Options(chain *yfinance.OptionChain) (calls dataframe.DataFrame, puts dataframe.DataFrame) {
	if chain == nil {
		return dataframe.LoadMaps(nil), dataframe.LoadMaps(nil)
	}
	return OptionContracts(chain.Calls), OptionContracts(chain.Puts)
}

// OptionContracts converts option contracts into a DataFrame using Yahoo's
// option-chain field names.
func OptionContracts(contracts []yfinance.OptionContract) dataframe.DataFrame {
	rows := make([]map[string]any, 0, len(contracts))
	for _, contract := range contracts {
		rows = append(rows, map[string]any{
			"contractSymbol":    contract.ContractSymbol,
			"lastTradeDate":     formatTime(contract.LastTradeDate),
			"strike":            contract.Strike,
			"lastPrice":         contract.LastPrice,
			"bid":               contract.Bid,
			"ask":               contract.Ask,
			"change":            contract.Change,
			"percentChange":     contract.PercentChange,
			"volume":            contract.Volume,
			"openInterest":      contract.OpenInterest,
			"impliedVolatility": contract.ImpliedVolatility,
			"inTheMoney":        contract.InTheMoney,
			"contractSize":      contract.ContractSize,
			"currency":          contract.Currency,
			"expiration":        formatTime(contract.Expiration),
		})
	}
	return dataframe.LoadMaps(rows)
}

// SearchResults converts search quote results into a DataFrame.
func SearchResults(results []yfinance.SearchResult) dataframe.DataFrame {
	rows := make([]map[string]any, 0, len(results))
	for _, result := range results {
		rows = append(rows, map[string]any{
			"symbol":         result.Symbol,
			"shortName":      result.ShortName,
			"longName":       result.LongName,
			"quoteType":      result.QuoteType,
			"exchange":       result.Exchange,
			"score":          result.Score,
			"typeDisp":       result.TypeDisp,
			"exchangeDisp":   result.ExchangeDisp,
			"sector":         result.Sector,
			"industry":       result.Industry,
			"isYahooFinance": result.IsYahooFinance,
		})
	}
	return dataframe.LoadMaps(rows)
}

// Search converts a SearchResponse's quote results into a DataFrame.
func Search(response *yfinance.SearchResponse) dataframe.DataFrame {
	if response == nil {
		return dataframe.LoadMaps(nil)
	}
	return SearchResults(response.Quotes)
}

// QuoteSummary converts raw quoteSummary modules into a long-form DataFrame
// with Module, Key, and Value columns.
func QuoteSummary(summary map[string]any) dataframe.DataFrame {
	rows := make([]map[string]any, 0)
	modules := sortedKeys(summary)
	for _, module := range modules {
		value := summary[module]
		obj, ok := value.(map[string]any)
		if !ok {
			rows = append(rows, map[string]any{
				"Module": module,
				"Key":    "",
				"Value":  scalarString(value),
			})
			continue
		}
		for _, key := range sortedKeys(obj) {
			rows = append(rows, map[string]any{
				"Module": module,
				"Key":    key,
				"Value":  scalarString(unwrapYahooValue(obj[key])),
			})
		}
	}
	return dataframe.LoadMaps(rows)
}

// KeyValues converts a single map into a two-column Key/Value DataFrame.
func KeyValues(values map[string]any) dataframe.DataFrame {
	rows := make([]map[string]any, 0, len(values))
	for _, key := range sortedKeys(values) {
		rows = append(rows, map[string]any{
			"Key":   key,
			"Value": scalarString(unwrapYahooValue(values[key])),
		})
	}
	return dataframe.LoadMaps(rows)
}

// Records converts map records into a DataFrame after unwrapping Yahoo
// {raw, fmt} values and stringifying nested values that Gota cannot store as
// scalar cells.
func Records(records []map[string]any) dataframe.DataFrame {
	rows := make([]map[string]any, 0, len(records))
	for _, record := range records {
		row := make(map[string]any, len(record))
		for key, value := range record {
			row[key] = scalarValue(unwrapYahooValue(value))
		}
		rows = append(rows, row)
	}
	return dataframe.LoadMaps(rows)
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func unwrapYahooValue(value any) any {
	obj, ok := value.(map[string]any)
	if !ok {
		return value
	}
	if raw, ok := obj["raw"]; ok {
		return raw
	}
	if formatted, ok := obj["fmt"]; ok {
		return formatted
	}
	return value
}

func scalarValue(value any) any {
	switch v := value.(type) {
	case nil, string, bool, int, int64, float64, float32:
		return v
	case time.Time:
		return formatTime(v)
	default:
		return scalarString(v)
	}
}

func scalarString(value any) string {
	switch v := scalarValueNoJSON(value).(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func scalarValueNoJSON(value any) any {
	switch v := value.(type) {
	case nil, string, bool, int, int64, float64, float32:
		return v
	case time.Time:
		return formatTime(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}

func finiteOrNaN(value float64) float64 {
	if math.IsInf(value, 0) {
		return math.NaN()
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
