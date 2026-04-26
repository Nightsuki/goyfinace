// Package yfinance provides a Go client for public Yahoo Finance endpoints.
//
// The package mirrors the common yfinance workflows for downloading historical
// prices, reading quote metadata, searching symbols, fetching option chains, and
// retrieving quote-summary financial modules. Yahoo Finance does not publish a
// supported public API for these endpoints, so callers should handle upstream
// response changes and rate limiting as normal operational concerns.
package yfinance
