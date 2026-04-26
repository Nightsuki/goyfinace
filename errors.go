package yfinance

import (
	"errors"
	"fmt"
)

var (
	// ErrNoResult indicates Yahoo returned a syntactically valid response that
	// did not contain a result for the requested symbol.
	ErrNoResult = errors.New("yfinance: no result")

	// ErrRateLimited indicates Yahoo rejected the request due to rate limiting.
	ErrRateLimited = errors.New("yfinance: rate limited")
)

// YahooError represents an error object returned by Yahoo Finance.
type YahooError struct {
	Code        string
	Description string
}

func (e *YahooError) Error() string {
	if e == nil {
		return "yfinance: yahoo error"
	}
	if e.Code == "" {
		return e.Description
	}
	if e.Description == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}
