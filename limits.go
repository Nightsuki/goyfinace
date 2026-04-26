package yfinance

import "fmt"

const (
	maxQuoteSymbols    = 100
	maxLookupCount     = 100
	maxScreenerCount   = 250
	maxNewsCount       = 100
	maxCalendarSize    = 100
	maxTimeseriesTypes = 256
)

func validateMax(name string, value int, max int) error {
	if value > max {
		return fmt.Errorf("yfinance: %s %d exceeds maximum %d", name, value, max)
	}
	return nil
}
