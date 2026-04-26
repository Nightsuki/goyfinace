package gota

import (
	"math"
	"testing"
	"time"

	yfinance "github.com/Nightsuki/goyfinace"
)

func TestCandlesDataFrame(t *testing.T) {
	df := Candles([]yfinance.Candle{
		{
			Time:         time.Unix(1700000000, 0).UTC(),
			Open:         1,
			High:         2,
			Low:          0.5,
			Close:        1.5,
			AdjClose:     1.4,
			Volume:       100,
			Dividends:    0.1,
			CapitalGains: 0.2,
			Split:        4,
		},
		{
			Time:  time.Unix(1700086400, 0).UTC(),
			Open:  math.NaN(),
			Close: 2,
		},
	})
	if df.Nrow() != 2 {
		t.Fatalf("Nrow = %d", df.Nrow())
	}
	wantColumns := []string{"Adj Close", "Capital Gains", "Close", "Dividends", "High", "Low", "Open", "Stock Splits", "Time", "Timestamp", "Volume"}
	if got := df.Names(); !sameStringSet(got, wantColumns) {
		t.Fatalf("columns = %v", got)
	}
	if got := df.Col("Close").Float()[0]; got != 1.5 {
		t.Fatalf("Close[0] = %v", got)
	}
}

func TestActionsDataFrame(t *testing.T) {
	df := Actions([]yfinance.Action{{
		Time:  time.Unix(1700000000, 0).UTC(),
		Type:  yfinance.ActionDividend,
		Value: 0.24,
	}})
	if df.Nrow() != 1 {
		t.Fatalf("Nrow = %d", df.Nrow())
	}
	if got := df.Col("Type").Records()[0]; got != "dividend" {
		t.Fatalf("Type = %q", got)
	}
}

func TestOptionsDataFrames(t *testing.T) {
	calls, puts := Options(&yfinance.OptionChain{
		Calls: []yfinance.OptionContract{{
			ContractSymbol:    "AAPL240119C00100000",
			Strike:            100,
			LastPrice:         3.5,
			ImpliedVolatility: 0.25,
			InTheMoney:        true,
			Expiration:        time.Unix(1700000000, 0).UTC(),
		}},
		Puts: []yfinance.OptionContract{{
			ContractSymbol: "AAPL240119P00100000",
			Strike:         100,
		}},
	})
	if calls.Nrow() != 1 || puts.Nrow() != 1 {
		t.Fatalf("rows calls=%d puts=%d", calls.Nrow(), puts.Nrow())
	}
	if got := calls.Col("contractSymbol").Records()[0]; got != "AAPL240119C00100000" {
		t.Fatalf("contractSymbol = %q", got)
	}
}

func TestQuoteSummaryDataFrameUnwrapsYahooValues(t *testing.T) {
	df := QuoteSummary(map[string]any{
		"price": map[string]any{
			"symbol":             "AAPL",
			"regularMarketPrice": map[string]any{"raw": 201.5, "fmt": "201.50"},
		},
	})
	if df.Nrow() != 2 {
		t.Fatalf("Nrow = %d", df.Nrow())
	}
	values := df.Col("Value").Records()
	if !contains(values, "201.5") || !contains(values, "AAPL") {
		t.Fatalf("values = %v", values)
	}
}

func TestRecordsStringifiesNestedValues(t *testing.T) {
	df := Records([]map[string]any{{
		"symbol": "AAPL",
		"price":  map[string]any{"raw": 201.5, "fmt": "201.50"},
		"nested": map[string]any{"x": 1},
	}})
	if df.Nrow() != 1 {
		t.Fatalf("Nrow = %d", df.Nrow())
	}
	if got := df.Col("price").Float()[0]; got != 201.5 {
		t.Fatalf("price = %v", got)
	}
	if got := df.Col("nested").Records()[0]; got == "" {
		t.Fatal("nested value should be stringified")
	}
}

func TestTimeseriesDataFrame(t *testing.T) {
	df := Timeseries(map[string]any{
		"timeseries": map[string]any{
			"result": []any{map[string]any{
				"quarterlyNetIncome": []any{map[string]any{
					"asOfDate":      "2023-09-30",
					"reportedValue": map[string]any{"raw": 50.0},
				}},
				"annualTotalRevenue": []any{map[string]any{
					"asOfDate":     "2023-12-31",
					"periodType":   "12M",
					"currencyCode": "USD",
					"reportedValue": map[string]any{
						"raw": 100.0,
						"fmt": "100",
					},
				}},
			}},
		},
	})
	if df.Nrow() != 2 {
		t.Fatalf("Nrow = %d", df.Nrow())
	}
	if got := df.Col("Type").Records()[0]; got != "annualTotalRevenue" {
		t.Fatalf("Type = %q", got)
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
