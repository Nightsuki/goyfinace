package yfinance

import (
	"context"
	"net/url"
)

// SectorData contains typed views over a Yahoo /v1/finance/sectors response.
//
// The structured accessors expose the same fields that yfinance's
// `Sector` class surfaces (overview, top_companies, top_etfs,
// top_mutual_funds, industries, top_growth_companies, top_performing_companies).
// Use Raw for fields not promoted by this struct.
type SectorData struct {
	Key  string
	Name string
	Raw  map[string]any
}

// Overview returns the sector's overview block ("description", "marketCap",
// "marketWeight", "employeeCount", "companiesCount", etc.).
func (s *SectorData) Overview() map[string]any {
	if s == nil {
		return nil
	}
	return mapAt(s.Raw, "data", "overview")
}

// TopCompanies returns the rows of the sector's top-companies table.
func (s *SectorData) TopCompanies() []map[string]any {
	return rowsAt(s.Raw, "data", "topCompanies")
}

// TopETFs returns the rows of the sector's top-ETFs table.
func (s *SectorData) TopETFs() []map[string]any {
	return rowsAt(s.Raw, "data", "topETFs")
}

// TopMutualFunds returns the rows of the sector's top-mutual-funds table.
func (s *SectorData) TopMutualFunds() []map[string]any {
	return rowsAt(s.Raw, "data", "topMutualFunds")
}

// Industries returns the sector's industries breakdown.
func (s *SectorData) Industries() []map[string]any {
	return rowsAt(s.Raw, "data", "industries")
}

// TopGrowthCompanies returns the sector's top-growth-companies table.
func (s *SectorData) TopGrowthCompanies() []map[string]any {
	return rowsAt(s.Raw, "data", "topGrowthCompanies")
}

// TopPerformingCompanies returns the sector's top-performing-companies table.
func (s *SectorData) TopPerformingCompanies() []map[string]any {
	return rowsAt(s.Raw, "data", "topPerformingCompanies")
}

// IndustryData contains typed views over a Yahoo /v1/finance/industries response.
type IndustryData struct {
	Key        string
	Name       string
	SectorKey  string
	SectorName string
	Raw        map[string]any
}

// Overview returns the industry's overview block.
func (i *IndustryData) Overview() map[string]any {
	if i == nil {
		return nil
	}
	return mapAt(i.Raw, "data", "overview")
}

// TopPerformingCompanies returns the industry's top-performing-companies rows.
func (i *IndustryData) TopPerformingCompanies() []map[string]any {
	return rowsAt(i.Raw, "data", "topPerformingCompanies")
}

// TopGrowthCompanies returns the industry's top-growth-companies rows.
func (i *IndustryData) TopGrowthCompanies() []map[string]any {
	return rowsAt(i.Raw, "data", "topGrowthCompanies")
}

// KeyCompanyKeys returns the industry's `key_company_keys` symbol list.
func (i *IndustryData) KeyCompanyKeys() []string {
	return stringsAt(i.Raw, "data", "keyCompanyKeys")
}

// KeyCompanyGroups returns the industry's labelled company groups.
func (i *IndustryData) KeyCompanyGroups() []map[string]any {
	return rowsAt(i.Raw, "data", "keyCompanyGroups")
}

// SectorOf fetches Yahoo sector data by key as a typed SectorData value.
func (c *Client) SectorOf(ctx context.Context, key string) (*SectorData, error) {
	raw, err := c.domain(ctx, "/v1/finance/sectors/"+url.PathEscape(key))
	if err != nil {
		return nil, err
	}
	out := &SectorData{Key: key, Raw: raw}
	if overview := mapAt(raw, "data", "overview"); overview != nil {
		out.Name = stringValue(overview["name"])
	}
	return out, nil
}

// IndustryOf fetches Yahoo industry data by key as a typed IndustryData value.
func (c *Client) IndustryOf(ctx context.Context, key string) (*IndustryData, error) {
	raw, err := c.domain(ctx, "/v1/finance/industries/"+url.PathEscape(key))
	if err != nil {
		return nil, err
	}
	out := &IndustryData{Key: key, Raw: raw}
	if overview := mapAt(raw, "data", "overview"); overview != nil {
		out.Name = stringValue(overview["name"])
		out.SectorKey = stringValue(overview["sectorKey"])
		out.SectorName = stringValue(overview["sectorName"])
	}
	return out, nil
}

func mapAt(root any, path ...string) map[string]any {
	cur := root
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[key]
	}
	out, _ := cur.(map[string]any)
	return out
}

func rowsAt(root any, path ...string) []map[string]any {
	cur := root
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[key]
	}
	switch v := cur.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if obj, ok := item.(map[string]any); ok {
				out = append(out, obj)
			}
		}
		return out
	case []map[string]any:
		return v
	}
	return nil
}

func stringsAt(root any, path ...string) []string {
	cur := root
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[key]
	}
	arr, ok := cur.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := stringValue(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}
