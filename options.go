package yfinance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// OptionChain contains calls and puts for one expiration date.
type OptionChain struct {
	Symbol         string
	Underlying     map[string]any
	ExpirationDate time.Time
	Expirations    []time.Time
	Strikes        []float64
	Calls          []OptionContract
	Puts           []OptionContract
}

// OptionContract is an option row returned by Yahoo.
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

// Options fetches an option chain. Pass a zero expiration to fetch Yahoo's default expiration.
func (t *Ticker) Options(ctx context.Context, expiration time.Time) (*OptionChain, error) {
	return t.c().Options(ctx, t.Symbol, expiration)
}

// Options fetches an option chain. Pass a zero expiration to fetch Yahoo's default expiration.
func (c *Client) Options(ctx context.Context, symbol string, expiration time.Time) (*OptionChain, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("yfinance: empty symbol")
	}
	q := url.Values{}
	if !expiration.IsZero() {
		q.Set("date", strconv.FormatInt(expiration.Unix(), 10))
	}
	var resp optionsResponse
	if err := c.getJSON(ctx, c.cloneWithDefaults().Query2URL, "/v7/finance/options/"+url.PathEscape(symbol), q, &resp); err != nil {
		return nil, err
	}
	if resp.OptionChain.Error != nil {
		return nil, resp.OptionChain.Error
	}
	if len(resp.OptionChain.Result) == 0 {
		return nil, ErrNoResult
	}
	result := resp.OptionChain.Result[0]
	out := &OptionChain{
		Symbol:     result.UnderlyingSymbol,
		Underlying: result.Quote,
		Strikes:    result.Strikes,
	}
	for _, ts := range result.ExpirationDates {
		out.Expirations = append(out.Expirations, time.Unix(ts, 0).UTC())
	}
	if len(result.Options) > 0 {
		opt := result.Options[0]
		out.ExpirationDate = time.Unix(opt.ExpirationDate, 0).UTC()
		out.Calls = decodeContracts(opt.Calls)
		out.Puts = decodeContracts(opt.Puts)
	}
	return out, nil
}

func decodeContracts(values []map[string]any) []OptionContract {
	out := make([]OptionContract, 0, len(values))
	for _, obj := range values {
		row := OptionContract{
			ContractSymbol:    stringValue(obj["contractSymbol"]),
			Strike:            numberValue(obj["strike"]),
			Currency:          stringValue(obj["currency"]),
			LastPrice:         numberValue(obj["lastPrice"]),
			Change:            numberValue(obj["change"]),
			PercentChange:     numberValue(obj["percentChange"]),
			Volume:            int64(numberValue(obj["volume"])),
			OpenInterest:      int64(numberValue(obj["openInterest"])),
			Bid:               numberValue(obj["bid"]),
			Ask:               numberValue(obj["ask"]),
			ContractSize:      stringValue(obj["contractSize"]),
			ImpliedVolatility: numberValue(obj["impliedVolatility"]),
			InTheMoney:        boolValue(obj["inTheMoney"]),
		}
		if ts := int64(numberValue(obj["expiration"])); ts > 0 {
			row.Expiration = time.Unix(ts, 0).UTC()
		}
		if ts := int64(numberValue(obj["lastTradeDate"])); ts > 0 {
			row.LastTradeDate = time.Unix(ts, 0).UTC()
		}
		out = append(out, row)
	}
	return out
}
