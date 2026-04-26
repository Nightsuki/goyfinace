package yfinance

import "context"

var financialModules = []string{
	"incomeStatementHistory",
	"incomeStatementHistoryQuarterly",
	"balanceSheetHistory",
	"balanceSheetHistoryQuarterly",
	"cashflowStatementHistory",
	"cashflowStatementHistoryQuarterly",
	"earnings",
	"earningsTrend",
}

// Financials fetches Yahoo quoteSummary financial statement modules.
func (t *Ticker) Financials(ctx context.Context) (map[string]any, error) {
	return t.c().Financials(ctx, t.Symbol)
}

// Financials fetches Yahoo quoteSummary financial statement modules.
func (c *Client) Financials(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, financialModules...)
}

// Recommendations fetches analyst recommendation modules.
func (t *Ticker) Recommendations(ctx context.Context) (map[string]any, error) {
	return t.c().Recommendations(ctx, t.Symbol)
}

// Recommendations fetches analyst recommendation modules.
func (c *Client) Recommendations(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "recommendationTrend", "upgradeDowngradeHistory")
}

// Holders fetches institutional, fund, and insider holders modules.
func (t *Ticker) Holders(ctx context.Context) (map[string]any, error) {
	return t.c().Holders(ctx, t.Symbol)
}

// Holders fetches institutional, fund, and insider holders modules.
func (c *Client) Holders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "institutionOwnership", "fundOwnership", "majorHoldersBreakdown", "insiderHolders")
}
