package yfinance_test

import (
	"context"
	"fmt"
	"log"

	yfinance "github.com/Nightsuki/goyfinace"
)

func ExampleClient_History() {
	client := yfinance.NewClient(nil)
	history, err := client.History(context.Background(), "AAPL", yfinance.HistoryParams{
		Period:   yfinance.Period1Mo,
		Interval: yfinance.Interval1D,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(history.Symbol)
}
