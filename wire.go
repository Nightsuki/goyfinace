package yfinance

import (
	"encoding/json"
	"math"
	"strconv"
)

type jsonNumber struct {
	Valid bool
	Value json.Number
}

func (n *jsonNumber) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	var value json.Number
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Valid = true
	n.Value = value
	return nil
}

func (n jsonNumber) Float64() float64 {
	if !n.Valid {
		return math.NaN()
	}
	v, err := n.Value.Float64()
	if err != nil {
		return math.NaN()
	}
	return v
}

func (n jsonNumber) Int64() int64 {
	if !n.Valid {
		return 0
	}
	v, err := strconv.ParseInt(n.Value.String(), 10, 64)
	if err == nil {
		return v
	}
	f, err := n.Value.Float64()
	if err != nil {
		return 0
	}
	return int64(f)
}

type chartResponse struct {
	Chart struct {
		Result []chartResult `json:"result"`
		Error  *YahooError   `json:"error"`
	} `json:"chart"`
}

type chartResult struct {
	Meta       map[string]any `json:"meta"`
	Timestamp  []int64        `json:"timestamp"`
	Indicators struct {
		Quote    []chartQuote `json:"quote"`
		AdjClose []struct {
			AdjClose []jsonNumber `json:"adjclose"`
		} `json:"adjclose"`
	} `json:"indicators"`
	Events struct {
		Dividends map[string]struct {
			Amount jsonNumber `json:"amount"`
			Date   int64      `json:"date"`
		} `json:"dividends"`
		Splits map[string]struct {
			Date        int64      `json:"date"`
			Numerator   jsonNumber `json:"numerator"`
			Denominator jsonNumber `json:"denominator"`
			SplitRatio  string     `json:"splitRatio"`
		} `json:"splits"`
	} `json:"events"`
}

type chartQuote struct {
	Open   []jsonNumber `json:"open"`
	High   []jsonNumber `json:"high"`
	Low    []jsonNumber `json:"low"`
	Close  []jsonNumber `json:"close"`
	Volume []jsonNumber `json:"volume"`
}

type quoteSummaryResponse struct {
	QuoteSummary struct {
		Result []map[string]any `json:"result"`
		Error  *YahooError      `json:"error"`
	} `json:"quoteSummary"`
}

type optionsResponse struct {
	OptionChain struct {
		Result []struct {
			UnderlyingSymbol string           `json:"underlyingSymbol"`
			ExpirationDates  []int64          `json:"expirationDates"`
			Strikes          []float64        `json:"strikes"`
			Quote            map[string]any   `json:"quote"`
			Options          []optionResponse `json:"options"`
		} `json:"result"`
		Error *YahooError `json:"error"`
	} `json:"optionChain"`
}

type optionResponse struct {
	ExpirationDate int64            `json:"expirationDate"`
	Calls          []map[string]any `json:"calls"`
	Puts           []map[string]any `json:"puts"`
}

func unwrapYahooValue(value any) any {
	obj, ok := value.(map[string]any)
	if !ok {
		return value
	}
	if raw, ok := obj["raw"]; ok {
		return raw
	}
	if fmtValue, ok := obj["fmt"]; ok {
		return fmtValue
	}
	return value
}

func stringValue(value any) string {
	switch v := unwrapYahooValue(value).(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func numberValue(value any) float64 {
	switch v := unwrapYahooValue(value).(type) {
	case json.Number:
		f, _ := v.Float64()
		return f
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	v, ok := unwrapYahooValue(value).(bool)
	return ok && v
}
