package yfinance

import "context"

// FundsData mirrors yfinance's `FundsData` accessor: a typed view over the
// fundProfile/topHoldings/quoteType/summaryProfile modules of an ETF or
// mutual fund.
type FundsData struct {
	Symbol string
	Raw    map[string]any
}

// FundsData fetches fund-specific quoteSummary modules and returns a typed wrapper.
func (c *Client) FundsData(ctx context.Context, symbol string) (*FundsData, error) {
	raw, err := c.FundProfile(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return &FundsData{Symbol: normalizeSymbol(symbol), Raw: raw}, nil
}

// FundsData fetches fund-specific quoteSummary modules for this ticker.
func (t *Ticker) FundsData(ctx context.Context) (*FundsData, error) {
	return t.c().FundsData(ctx, t.Symbol)
}

// Description returns the fund prospectus description.
func (f *FundsData) Description() string {
	if f == nil {
		return ""
	}
	if profile := mapAt(f.Raw, "summaryProfile"); profile != nil {
		if s := stringValue(profile["longBusinessSummary"]); s != "" {
			return s
		}
	}
	if profile := mapAt(f.Raw, "assetProfile"); profile != nil {
		return stringValue(profile["longBusinessSummary"])
	}
	return ""
}

// FundOverview returns the typed overview ("family", "category", "legalType").
func (f *FundsData) FundOverview() map[string]any {
	if f == nil {
		return nil
	}
	profile := mapAt(f.Raw, "fundProfile")
	if profile == nil {
		return nil
	}
	return map[string]any{
		"family":    stringValue(profile["family"]),
		"category":  stringValue(profile["categoryName"]),
		"legalType": stringValue(profile["legalType"]),
	}
}

// FundOperations returns the fund-operations table from fundProfile.
func (f *FundsData) FundOperations() map[string]any {
	if f == nil {
		return nil
	}
	return mapAt(f.Raw, "fundProfile", "feesExpensesInvestment")
}

// AssetClasses returns the topHoldings asset-class breakdown
// (cashPosition, stockPosition, bondPosition, etc.).
func (f *FundsData) AssetClasses() map[string]any {
	if f == nil {
		return nil
	}
	holdings := mapAt(f.Raw, "topHoldings")
	if holdings == nil {
		return nil
	}
	out := make(map[string]any)
	for _, key := range []string{
		"cashPosition", "stockPosition", "bondPosition",
		"preferredPosition", "convertiblePosition", "otherPosition",
	} {
		if v, ok := holdings[key]; ok {
			out[key] = unwrapYahooValue(v)
		}
	}
	return out
}

// TopHoldings returns the topHoldings.holdings rows.
func (f *FundsData) TopHoldings() []map[string]any {
	return rowsAt(f.Raw, "topHoldings", "holdings")
}

// EquityHoldings returns the topHoldings.equityHoldings statistics.
func (f *FundsData) EquityHoldings() map[string]any {
	return mapAt(f.Raw, "topHoldings", "equityHoldings")
}

// BondHoldings returns the topHoldings.bondHoldings statistics.
func (f *FundsData) BondHoldings() map[string]any {
	return mapAt(f.Raw, "topHoldings", "bondHoldings")
}

// BondRatings returns the topHoldings.bondRatings rows.
func (f *FundsData) BondRatings() []map[string]any {
	return rowsAt(f.Raw, "topHoldings", "bondRatings")
}

// SectorWeightings returns the topHoldings.sectorWeightings rows.
func (f *FundsData) SectorWeightings() []map[string]any {
	return rowsAt(f.Raw, "topHoldings", "sectorWeightings")
}
