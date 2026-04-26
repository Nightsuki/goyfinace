package gota_test

import (
	"fmt"
	"time"

	yfinance "github.com/Nightsuki/goyfinace"
	yfgota "github.com/Nightsuki/goyfinace/adapter/gota"
)

func ExampleHistory() {
	history := &yfinance.HistoryResult{
		Symbol: "AAPL",
		Candles: []yfinance.Candle{{
			Time:     time.Unix(1700000000, 0).UTC(),
			Open:     190,
			High:     201,
			Low:      189,
			Close:    200,
			AdjClose: 199,
			Volume:   1000,
		}},
	}

	df := yfgota.History(history)
	fmt.Println(df.Nrow(), df.Col("Close").Float()[0])
	// Output: 1 200
}
