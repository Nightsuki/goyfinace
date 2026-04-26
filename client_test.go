package yfinance

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHistoryDecodesCandlesAndEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/AAPL" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("range"); got != "5d" {
			t.Fatalf("range = %q", got)
		}
		if got := r.URL.Query().Get("interval"); got != "1d" {
			t.Fatalf("interval = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta": map[string]any{
						"currency":             "USD",
						"symbol":               "AAPL",
						"exchangeName":         "NMS",
						"exchangeTimezoneName": "America/New_York",
						"regularMarketPrice":   200.25,
						"chartPreviousClose":   199.5,
					},
					"timestamp": []int64{1700000000, 1700086400},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open":   []any{190.0, nil},
							"high":   []any{201.0, 202.0},
							"low":    []any{189.0, 198.0},
							"close":  []any{200.0, 201.0},
							"volume": []any{1000, 2000},
						}},
						"adjclose": []any{map[string]any{"adjclose": []any{199.0, 200.0}}},
					},
					"events": map[string]any{
						"dividends": map[string]any{
							"1700000000": map[string]any{"amount": 0.24, "date": 1700000000},
						},
						"splits": map[string]any{
							"1700086400": map[string]any{"numerator": 4, "denominator": 1, "date": 1700086400},
						},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Ticker("aapl").History(context.Background(), HistoryParams{
		Period:   Period5D,
		Interval: Interval1D,
	})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if result.Meta.Currency != "USD" || result.Meta.Symbol != "AAPL" {
		t.Fatalf("unexpected metadata: %+v", result.Meta)
	}
	if len(result.Candles) != 2 {
		t.Fatalf("len(candles) = %d", len(result.Candles))
	}
	if result.Candles[0].Dividends != 0.24 {
		t.Fatalf("dividend = %v", result.Candles[0].Dividends)
	}
	if result.Candles[1].Split != 4 {
		t.Fatalf("split = %v", result.Candles[1].Split)
	}
	if !math.IsNaN(result.Candles[1].Open) {
		t.Fatalf("expected null open to decode as NaN, got %v", result.Candles[1].Open)
	}
}

func TestHistoryDateRangeUsesPeriodUnixParams(t *testing.T) {
	start := time.Unix(1700000000, 0).UTC()
	end := start.Add(24 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("period1"); got != "1700000000" {
			t.Fatalf("period1 = %q", got)
		}
		if got := r.URL.Query().Get("period2"); got != "1700086400" {
			t.Fatalf("period2 = %q", got)
		}
		if got := r.URL.Query().Get("range"); got != "" {
			t.Fatalf("range should be empty, got %q", got)
		}
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	_, err := client.History(context.Background(), "MSFT", HistoryParams{Start: start, End: end})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
}

func TestInfoFlattensYahooRawValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v10/finance/quoteSummary/AAPL" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Query().Get("modules"), "price") {
			t.Fatalf("modules missing price: %s", r.URL.RawQuery)
		}
		writeJSON(t, w, map[string]any{
			"quoteSummary": map[string]any{
				"result": []any{map[string]any{
					"price": map[string]any{
						"symbol":             "AAPL",
						"regularMarketPrice": map[string]any{"raw": 201.5, "fmt": "201.50"},
					},
					"quoteType": map[string]any{
						"quoteType": "EQUITY",
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	info, err := client.Info(context.Background(), "aapl")
	if err != nil {
		t.Fatalf("Info returned error: %v", err)
	}
	if info["symbol"] != "AAPL" {
		t.Fatalf("symbol = %#v", info["symbol"])
	}
	if got := numberValue(info["regularMarketPrice"]); got != 201.5 {
		t.Fatalf("regularMarketPrice = %v", got)
	}
}

func TestOptionsDecodesContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("date"); got != "1700000000" {
			t.Fatalf("date = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"optionChain": map[string]any{
				"result": []any{map[string]any{
					"underlyingSymbol": "AAPL",
					"expirationDates":  []int64{1700000000},
					"strikes":          []float64{200},
					"quote":            map[string]any{"symbol": "AAPL"},
					"options": []any{map[string]any{
						"expirationDate": int64(1700000000),
						"calls": []any{map[string]any{
							"contractSymbol":    "AAPL231114C00200000",
							"strike":            200,
							"lastPrice":         3.5,
							"lastTradeDate":     1699990000,
							"expiration":        1700000000,
							"impliedVolatility": 0.2,
							"inTheMoney":        true,
						}},
						"puts": []any{},
					}},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	chain, err := client.Options(context.Background(), "aapl", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("Options returned error: %v", err)
	}
	if chain.Symbol != "AAPL" || len(chain.Calls) != 1 {
		t.Fatalf("unexpected chain: %+v", chain)
	}
	if !chain.Calls[0].InTheMoney || chain.Calls[0].Strike != 200 {
		t.Fatalf("unexpected call: %+v", chain.Calls[0])
	}
}

func TestSearchDecodesQuotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/finance/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "apple" {
			t.Fatalf("q = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"quotes": []any{map[string]any{
				"symbol":         "AAPL",
				"shortname":      "Apple Inc.",
				"quoteType":      "EQUITY",
				"exchange":       "NMS",
				"isYahooFinance": true,
			}},
			"news": []any{},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Search(context.Background(), "apple", 5, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(result.Quotes) != 1 || result.Quotes[0].Symbol != "AAPL" {
		t.Fatalf("unexpected search result: %+v", result.Quotes)
	}
}

func TestDownloadRunsAllSymbols(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[strings.TrimPrefix(r.URL.Path, "/v8/finance/chart/")] = true
		mu.Unlock()
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	results := client.Download(context.Background(), []string{"aapl", "msft"}, HistoryParams{})
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	for _, result := range results {
		if result.Err != nil {
			t.Fatalf("%s returned error: %v", result.Symbol, result.Err)
		}
	}
	if !seen["AAPL"] || !seen["MSFT"] {
		t.Fatalf("seen = %+v", seen)
	}
}

func writeEmptyChart(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	writeJSON(t, w, map[string]any{
		"chart": map[string]any{
			"result": []any{map[string]any{
				"meta":       map[string]any{"symbol": "TEST"},
				"timestamp":  []int64{},
				"indicators": map[string]any{"quote": []any{map[string]any{}}},
			}},
			"error": nil,
		},
	})
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}
