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

// Earnings fetches Yahoo earnings modules.
func (t *Ticker) Earnings(ctx context.Context) (map[string]any, error) {
	return t.c().Earnings(ctx, t.Symbol)
}

// Earnings fetches Yahoo earnings modules.
func (c *Client) Earnings(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earnings", "earningsHistory", "earningsTrend")
}

// EarningsEstimate fetches earnings-estimate data from Yahoo earningsTrend.
func (t *Ticker) EarningsEstimate(ctx context.Context) (map[string]any, error) {
	return t.c().EarningsEstimate(ctx, t.Symbol)
}

// EarningsEstimate fetches earnings-estimate data from Yahoo earningsTrend.
func (c *Client) EarningsEstimate(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsTrend")
}

// RevenueEstimate fetches revenue-estimate data from Yahoo earningsTrend.
func (t *Ticker) RevenueEstimate(ctx context.Context) (map[string]any, error) {
	return t.c().RevenueEstimate(ctx, t.Symbol)
}

// RevenueEstimate fetches revenue-estimate data from Yahoo earningsTrend.
func (c *Client) RevenueEstimate(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsTrend")
}

// EarningsHistory fetches earnings-history data.
func (t *Ticker) EarningsHistory(ctx context.Context) (map[string]any, error) {
	return t.c().EarningsHistory(ctx, t.Symbol)
}

// EarningsHistory fetches earnings-history data.
func (c *Client) EarningsHistory(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsHistory")
}

// EPSRevisions fetches EPS revisions from Yahoo earningsTrend.
func (t *Ticker) EPSRevisions(ctx context.Context) (map[string]any, error) {
	return t.c().EPSRevisions(ctx, t.Symbol)
}

// EPSRevisions fetches EPS revisions from Yahoo earningsTrend.
func (c *Client) EPSRevisions(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsTrend")
}

// EPSTrend fetches EPS trend data from Yahoo earningsTrend.
func (t *Ticker) EPSTrend(ctx context.Context) (map[string]any, error) {
	return t.c().EPSTrend(ctx, t.Symbol)
}

// EPSTrend fetches EPS trend data from Yahoo earningsTrend.
func (c *Client) EPSTrend(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsTrend")
}

// GrowthEstimates fetches stock, industry, sector, and index growth estimates.
func (t *Ticker) GrowthEstimates(ctx context.Context) (map[string]any, error) {
	return t.c().GrowthEstimates(ctx, t.Symbol)
}

// GrowthEstimates fetches stock, industry, sector, and index growth estimates.
func (c *Client) GrowthEstimates(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "earningsTrend", "industryTrend", "sectorTrend", "indexTrend")
}

// Recommendations fetches analyst recommendation modules.
func (t *Ticker) Recommendations(ctx context.Context) (map[string]any, error) {
	return t.c().Recommendations(ctx, t.Symbol)
}

// Recommendations fetches analyst recommendation modules.
func (c *Client) Recommendations(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "recommendationTrend", "upgradeDowngradeHistory")
}

// RecommendationsSummary fetches Yahoo recommendation trend data.
func (t *Ticker) RecommendationsSummary(ctx context.Context) (map[string]any, error) {
	return t.c().RecommendationsSummary(ctx, t.Symbol)
}

// RecommendationsSummary fetches Yahoo recommendation trend data.
func (c *Client) RecommendationsSummary(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "recommendationTrend")
}

// Holders fetches institutional, fund, and insider holders modules.
func (t *Ticker) Holders(ctx context.Context) (map[string]any, error) {
	return t.c().Holders(ctx, t.Symbol)
}

// Holders fetches institutional, fund, and insider holders modules.
func (c *Client) Holders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "institutionOwnership", "fundOwnership", "majorHoldersBreakdown", "insiderHolders")
}

// MajorHolders fetches major-holders breakdown data.
func (t *Ticker) MajorHolders(ctx context.Context) (map[string]any, error) {
	return t.c().MajorHolders(ctx, t.Symbol)
}

// MajorHolders fetches major-holders breakdown data.
func (c *Client) MajorHolders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "majorHoldersBreakdown")
}

// InstitutionalHolders fetches institutional ownership data.
func (t *Ticker) InstitutionalHolders(ctx context.Context) (map[string]any, error) {
	return t.c().InstitutionalHolders(ctx, t.Symbol)
}

// InstitutionalHolders fetches institutional ownership data.
func (c *Client) InstitutionalHolders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "institutionOwnership")
}

// MutualFundHolders fetches fund ownership data.
func (t *Ticker) MutualFundHolders(ctx context.Context) (map[string]any, error) {
	return t.c().MutualFundHolders(ctx, t.Symbol)
}

// MutualFundHolders fetches fund ownership data.
func (c *Client) MutualFundHolders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "fundOwnership")
}

// InsiderPurchases fetches net share purchase activity.
func (t *Ticker) InsiderPurchases(ctx context.Context) (map[string]any, error) {
	return t.c().InsiderPurchases(ctx, t.Symbol)
}

// InsiderPurchases fetches net share purchase activity.
func (c *Client) InsiderPurchases(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "netSharePurchaseActivity")
}

// InsiderTransactions fetches insider transaction rows.
func (t *Ticker) InsiderTransactions(ctx context.Context) (map[string]any, error) {
	return t.c().InsiderTransactions(ctx, t.Symbol)
}

// InsiderTransactions fetches insider transaction rows.
func (c *Client) InsiderTransactions(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "insiderTransactions")
}

// InsiderRosterHolders fetches insider roster/holder rows.
func (t *Ticker) InsiderRosterHolders(ctx context.Context) (map[string]any, error) {
	return t.c().InsiderRosterHolders(ctx, t.Symbol)
}

// InsiderRosterHolders fetches insider roster/holder rows.
func (c *Client) InsiderRosterHolders(ctx context.Context, symbol string) (map[string]any, error) {
	return c.QuoteSummary(ctx, symbol, "insiderHolders")
}
