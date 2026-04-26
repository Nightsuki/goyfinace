package yfinance

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestQuoteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v7/finance/quote" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbols"); got != "AAPL,MSFT" {
			t.Fatalf("symbols = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"quoteResponse": map[string]any{
				"result": []any{map[string]any{
					"symbol":             "AAPL",
					"shortName":          "Apple Inc.",
					"quoteType":          "EQUITY",
					"regularMarketPrice": 201.5,
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	quotes, err := client.Quote(context.Background(), "aapl", "msft")
	if err != nil {
		t.Fatalf("Quote returned error: %v", err)
	}
	if len(quotes) != 1 || quotes[0].Symbol != "AAPL" || quotes[0].RegularMarketPrice != 201.5 {
		t.Fatalf("unexpected quotes: %+v", quotes)
	}
}

func TestTickersQuotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("symbols"); got != "AAPL,MSFT" {
			t.Fatalf("symbols = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"quoteResponse": map[string]any{
				"result": []any{map[string]any{"symbol": "AAPL"}, map[string]any{"symbol": "MSFT"}},
				"error":  nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	quotes, err := client.Tickers("aapl msft").Quotes(context.Background())
	if err != nil {
		t.Fatalf("Quotes returned error: %v", err)
	}
	if len(quotes) != 2 || quotes[0].Symbol != "AAPL" || quotes[1].Symbol != "MSFT" {
		t.Fatalf("quotes = %+v", quotes)
	}
}

func TestNewsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/xhr/ncp" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("queryRef"); got != "latestNews" {
			t.Fatalf("queryRef = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"data": map[string]any{
				"tickerStream": map[string]any{
					"stream": []any{
						map[string]any{"title": "real news"},
						map[string]any{"title": "ad", "ad": []any{"x"}},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.RootURL = server.URL
	news, err := client.News(context.Background(), "aapl", 5, "news")
	if err != nil {
		t.Fatalf("News returned error: %v", err)
	}
	if len(news) != 1 || news[0]["title"] != "real news" {
		t.Fatalf("unexpected news: %+v", news)
	}
}

func TestLookupScreenerAndMarketEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/finance/lookup":
			if got := r.URL.Query().Get("type"); got != "equity" {
				t.Fatalf("lookup type = %q", got)
			}
			writeJSON(t, w, map[string]any{"finance": map[string]any{"result": []any{map[string]any{"documents": []any{map[string]any{"symbol": "AAPL"}}}}}})
		case "/v1/finance/screener/predefined/saved":
			if got := r.URL.Query().Get("scrIds"); got != "day_gainers" {
				t.Fatalf("scrIds = %q", got)
			}
			writeJSON(t, w, map[string]any{"finance": map[string]any{"result": []any{map[string]any{"quotes": []any{}}}}})
		case "/v6/finance/quote/marketSummary":
			writeJSON(t, w, map[string]any{"marketSummaryResponse": map[string]any{"result": []any{}}})
		case "/v6/finance/markettime":
			writeJSON(t, w, map[string]any{"finance": map[string]any{"marketTimes": []any{}}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	if _, err := client.Lookup(context.Background(), "apple", LookupEquity, 3); err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if _, err := client.PredefinedScreen(context.Background(), "day_gainers", 5); err != nil {
		t.Fatalf("PredefinedScreen returned error: %v", err)
	}
	if _, err := client.MarketSummary(context.Background(), "us"); err != nil {
		t.Fatalf("MarketSummary returned error: %v", err)
	}
	if _, err := client.MarketStatus(context.Background(), "us"); err != nil {
		t.Fatalf("MarketStatus returned error: %v", err)
	}
}

func TestActionsIncludeCapitalGains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta":      map[string]any{"symbol": "AAPL"},
					"timestamp": []int64{1700000000},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open": []any{1}, "high": []any{1}, "low": []any{1}, "close": []any{1}, "volume": []any{1},
						}},
					},
					"events": map[string]any{
						"dividends":    map[string]any{"1700000000": map[string]any{"amount": 0.1}},
						"capitalGains": map[string]any{"1700000000": map[string]any{"amount": 0.2}},
						"splits":       map[string]any{"1700000000": map[string]any{"numerator": 4, "denominator": 1}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	actions, err := client.Actions(context.Background(), "AAPL", HistoryParams{Period: Period1Y})
	if err != nil {
		t.Fatalf("Actions returned error: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("actions = %+v", actions)
	}
}

func TestScreenCalendarAndDomainEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/finance/screener":
			if r.Method != http.MethodPost {
				t.Fatalf("screener method = %s", r.Method)
			}
			var body ScreenRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode screener body: %v", err)
			}
			if body.SortField != "ticker" || body.UserIDType != "guid" {
				t.Fatalf("unexpected screener defaults: %+v", body)
			}
			if body.Query != nil && body.Query["operator"] != "and" {
				t.Fatalf("unexpected query: %+v", body.Query)
			}
		case "/v1/finance/visualization":
			if r.Method != http.MethodPost {
				t.Fatalf("visualization method = %s", r.Method)
			}
			var body CalendarQuery
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode visualization body: %v", err)
			}
			if body.Size == 0 {
				t.Fatalf("expected visualization size default")
			}
		case "/v1/finance/sectors/technology", "/v1/finance/industries/software":
			if r.Method != http.MethodGet {
				t.Fatalf("domain method = %s", r.Method)
			}
		case "/v6/finance/quote/marketSummary":
			if r.Method != http.MethodGet {
				t.Fatalf("market summary method = %s", r.Method)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{"ok": true})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	if _, err := client.Screen(context.Background(), ScreenRequest{Query: EquityQuery(GT("intradaymarketcap", 1000))}); err != nil {
		t.Fatalf("Screen returned error: %v", err)
	}
	if _, err := client.CalendarVisualization(context.Background(), CalendarQuery{}); err != nil {
		t.Fatalf("CalendarVisualization returned error: %v", err)
	}
	if _, err := client.EarningsDates(context.Background(), "aapl", 0); err != nil {
		t.Fatalf("EarningsDates returned error: %v", err)
	}
	if _, err := client.Sector(context.Background(), "technology"); err != nil {
		t.Fatalf("Sector returned error: %v", err)
	}
	if _, err := client.Industry(context.Background(), "software"); err != nil {
		t.Fatalf("Industry returned error: %v", err)
	}
	if _, err := client.Market("us").Summary(context.Background()); err != nil {
		t.Fatalf("Market Summary returned error: %v", err)
	}
	if _, err := client.Calendars().Earnings(context.Background(), 5, "aapl"); err != nil {
		t.Fatalf("Calendars Earnings returned error: %v", err)
	}
}

func TestScreenerQueryBuilders(t *testing.T) {
	query := EquityQuery(GTE("intradaymarketcap", 1000), Between("eodvolume", 10, 20))
	if query["operator"] != "and" {
		t.Fatalf("operator = %#v", query["operator"])
	}
	operands, ok := query["operands"].([]any)
	if !ok || len(operands) != 3 {
		t.Fatalf("operands = %#v", query["operands"])
	}
	first, ok := operands[0].(map[string]any)
	if !ok || first["operator"] != "eq" {
		t.Fatalf("first operand = %#v", operands[0])
	}
}

func TestQuoteSummaryWrappersUseExpectedModules(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v10/finance/quoteSummary/AAPL" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls = append(calls, r.URL.Query().Get("modules"))
		writeJSON(t, w, map[string]any{
			"quoteSummary": map[string]any{
				"result": []any{map[string]any{"ok": map[string]any{}}},
				"error":  nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	ctx := context.Background()
	checks := []struct {
		name string
		call func() (map[string]any, error)
		want string
	}{
		{"Calendar", func() (map[string]any, error) { return client.Calendar(ctx, "AAPL") }, "calendarEvents"},
		{"SECFilings", func() (map[string]any, error) { return client.SECFilings(ctx, "AAPL") }, "secFilings"},
		{"Sustainability", func() (map[string]any, error) { return client.Sustainability(ctx, "AAPL") }, "esgScores"},
		{"Valuation", func() (map[string]any, error) { return client.Valuation(ctx, "AAPL") }, "financialData"},
		{"UpgradesDowngrades", func() (map[string]any, error) { return client.UpgradesDowngrades(ctx, "AAPL") }, "upgradeDowngradeHistory"},
		{"Analysis", func() (map[string]any, error) { return client.Analysis(ctx, "AAPL") }, "earningsTrend"},
		{"AnalystPriceTargets", func() (map[string]any, error) { return client.AnalystPriceTargets(ctx, "AAPL") }, "financialData"},
		{"FundProfile", func() (map[string]any, error) { return client.FundProfile(ctx, "AAPL") }, "fundProfile"},
		{"Earnings", func() (map[string]any, error) { return client.Earnings(ctx, "AAPL") }, "earningsHistory"},
		{"EarningsEstimate", func() (map[string]any, error) { return client.EarningsEstimate(ctx, "AAPL") }, "earningsTrend"},
		{"RevenueEstimate", func() (map[string]any, error) { return client.RevenueEstimate(ctx, "AAPL") }, "earningsTrend"},
		{"EarningsHistory", func() (map[string]any, error) { return client.EarningsHistory(ctx, "AAPL") }, "earningsHistory"},
		{"EPSRevisions", func() (map[string]any, error) { return client.EPSRevisions(ctx, "AAPL") }, "earningsTrend"},
		{"EPSTrend", func() (map[string]any, error) { return client.EPSTrend(ctx, "AAPL") }, "earningsTrend"},
		{"GrowthEstimates", func() (map[string]any, error) { return client.GrowthEstimates(ctx, "AAPL") }, "sectorTrend"},
		{"RecommendationsSummary", func() (map[string]any, error) { return client.RecommendationsSummary(ctx, "AAPL") }, "recommendationTrend"},
		{"MajorHolders", func() (map[string]any, error) { return client.MajorHolders(ctx, "AAPL") }, "majorHoldersBreakdown"},
		{"InstitutionalHolders", func() (map[string]any, error) { return client.InstitutionalHolders(ctx, "AAPL") }, "institutionOwnership"},
		{"MutualFundHolders", func() (map[string]any, error) { return client.MutualFundHolders(ctx, "AAPL") }, "fundOwnership"},
		{"InsiderPurchases", func() (map[string]any, error) { return client.InsiderPurchases(ctx, "AAPL") }, "netSharePurchaseActivity"},
		{"InsiderTransactions", func() (map[string]any, error) { return client.InsiderTransactions(ctx, "AAPL") }, "insiderTransactions"},
		{"InsiderRosterHolders", func() (map[string]any, error) { return client.InsiderRosterHolders(ctx, "AAPL") }, "insiderHolders"},
	}
	for i, check := range checks {
		if _, err := check.call(); err != nil {
			t.Fatalf("%s returned error: %v", check.name, err)
		}
		if !strings.Contains(calls[i], check.want) {
			t.Fatalf("%s modules = %q, want %q", check.name, calls[i], check.want)
		}
	}
}

func TestTimeseriesEndpoints(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/fundamentals-timeseries/v1/finance/timeseries/AAPL" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls = append(calls, r.URL.Query().Get("type"))
		if r.URL.Query().Get("period1") == "" || r.URL.Query().Get("period2") == "" {
			t.Fatalf("missing period query: %s", r.URL.RawQuery)
		}
		writeJSON(t, w, map[string]any{"timeseries": map[string]any{"result": []any{}}})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	ctx := context.Background()
	start := time.Unix(1700000000, 0)
	end := start.Add(24 * time.Hour)
	if _, err := client.FundamentalsTimeseries(ctx, "aapl", []string{"annualTotalRevenue"}, start, end); err != nil {
		t.Fatalf("FundamentalsTimeseries returned error: %v", err)
	}
	if _, err := client.SharesFull(ctx, "aapl", start, end); err != nil {
		t.Fatalf("SharesFull returned error: %v", err)
	}
	if _, err := client.IncomeStatement(ctx, "aapl", "quarterly"); err != nil {
		t.Fatalf("IncomeStatement returned error: %v", err)
	}
	if _, err := client.BalanceSheet(ctx, "aapl", "annual"); err != nil {
		t.Fatalf("BalanceSheet returned error: %v", err)
	}
	if _, err := client.CashFlow(ctx, "aapl", "ttm"); err != nil {
		t.Fatalf("CashFlow returned error: %v", err)
	}
	if !strings.Contains(calls[0], "annualTotalRevenue") || calls[1] != "shares_out" || !strings.Contains(calls[2], "quarterlyTotalRevenue") {
		t.Fatalf("unexpected timeseries calls: %+v", calls)
	}
}

func TestISINBestEffortExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v10/finance/quoteSummary/AAPL":
			writeJSON(t, w, map[string]any{
				"quoteSummary": map[string]any{
					"result": []any{map[string]any{"price": map[string]any{"shortName": "Apple Inc."}}},
					"error":  nil,
				},
			})
		case "/ajax/SearchController_Suggest":
			_, _ = w.Write([]byte(`"MSFT|US5949181045|Microsoft" "AAPL|US0378331005|Apple"`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	client.ISINURL = server.URL
	isin, err := client.ISIN(context.Background(), "aapl")
	if err != nil {
		t.Fatalf("ISIN returned error: %v", err)
	}
	if isin != "US0378331005" {
		t.Fatalf("isin = %q", isin)
	}
}

func TestISINRejectsAmbiguousSuggestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v10/finance/quoteSummary/AAPL":
			writeJSON(t, w, map[string]any{
				"quoteSummary": map[string]any{
					"result": []any{map[string]any{"price": map[string]any{"shortName": "Apple Inc."}}},
					"error":  nil,
				},
			})
		case "/ajax/SearchController_Suggest":
			_, _ = w.Write([]byte(`"MSFT|US5949181045|Microsoft"`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	client.ISINURL = server.URL
	if _, err := client.ISIN(context.Background(), "aapl"); !errors.Is(err, ErrNoResult) {
		t.Fatalf("ISIN error = %v, want ErrNoResult", err)
	}
}

func TestHistoryAutoAdjustAndRounding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta": map[string]any{
						"symbol":    "AAPL",
						"priceHint": 2,
					},
					"timestamp": []int64{1700000000, 1700086400},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open":   []any{100.0, 110.0},
							"high":   []any{105.0, 115.0},
							"low":    []any{95.0, 108.0},
							"close":  []any{100.0, 110.0},
							"volume": []any{1000, 2000},
						}},
						"adjclose": []any{map[string]any{"adjclose": []any{50.0, 110.0}}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Ticker("AAPL").History(context.Background(), HistoryParams{
		Period:     Period5D,
		AutoAdjust: true,
		Rounding:   true,
	})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	first := result.Candles[0]
	if first.Close != 50.0 {
		t.Fatalf("auto-adjusted close = %v, want 50", first.Close)
	}
	if first.Open != 50.0 || first.High != 52.5 || first.Low != 47.5 {
		t.Fatalf("auto-adjusted OHL = %v/%v/%v", first.Open, first.High, first.Low)
	}
	if first.Volume != 2000 {
		t.Fatalf("auto-adjusted volume = %v, want 2000", first.Volume)
	}
	second := result.Candles[1]
	if second.Open != 110.0 || second.Close != 110.0 {
		t.Fatalf("ratio-1 candle changed unexpectedly: %+v", second)
	}
}

func TestSectorIndustryTypedAccessors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/finance/sectors/technology":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"overview": map[string]any{
						"name":           "Technology",
						"marketCap":      1.0,
						"companiesCount": 500,
					},
					"topCompanies": []any{
						map[string]any{"symbol": "AAPL"},
						map[string]any{"symbol": "MSFT"},
					},
					"topETFs": []any{
						map[string]any{"symbol": "XLK"},
					},
					"topMutualFunds": []any{
						map[string]any{"symbol": "VITAX"},
					},
					"industries": []any{
						map[string]any{"key": "software-application"},
					},
					"topGrowthCompanies":     []any{map[string]any{"symbol": "NVDA"}},
					"topPerformingCompanies": []any{map[string]any{"symbol": "AVGO"}},
				},
			})
		case "/v1/finance/industries/software-application":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"overview": map[string]any{
						"name":       "Software—Application",
						"sectorKey":  "technology",
						"sectorName": "Technology",
					},
					"topPerformingCompanies": []any{map[string]any{"symbol": "ADBE"}},
					"topGrowthCompanies":     []any{map[string]any{"symbol": "CRM"}},
					"keyCompanyKeys":         []any{"ADBE", "CRM"},
					"keyCompanyGroups": []any{
						map[string]any{"name": "Leaders", "keys": []any{"ADBE", "CRM"}},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL

	sec, err := client.SectorOf(context.Background(), "technology")
	if err != nil {
		t.Fatalf("SectorOf: %v", err)
	}
	if sec.Name != "Technology" {
		t.Fatalf("Sector.Name = %q", sec.Name)
	}
	if got := sec.TopCompanies(); len(got) != 2 || stringValue(got[0]["symbol"]) != "AAPL" {
		t.Fatalf("TopCompanies = %+v", got)
	}
	if got := sec.TopETFs(); len(got) != 1 || stringValue(got[0]["symbol"]) != "XLK" {
		t.Fatalf("TopETFs = %+v", got)
	}
	if got := sec.TopMutualFunds(); len(got) != 1 {
		t.Fatalf("TopMutualFunds len = %d", len(got))
	}
	if got := sec.Industries(); len(got) != 1 {
		t.Fatalf("Industries len = %d", len(got))
	}
	if got := sec.TopGrowthCompanies(); len(got) != 1 {
		t.Fatalf("TopGrowthCompanies len = %d", len(got))
	}
	if got := sec.TopPerformingCompanies(); len(got) != 1 {
		t.Fatalf("TopPerformingCompanies len = %d", len(got))
	}
	if sec.Overview() == nil {
		t.Fatalf("Overview was nil")
	}

	ind, err := client.IndustryOf(context.Background(), "software-application")
	if err != nil {
		t.Fatalf("IndustryOf: %v", err)
	}
	if ind.SectorKey != "technology" {
		t.Fatalf("Industry.SectorKey = %q", ind.SectorKey)
	}
	if got := ind.KeyCompanyKeys(); len(got) != 2 || got[0] != "ADBE" {
		t.Fatalf("KeyCompanyKeys = %+v", got)
	}
	if got := ind.KeyCompanyGroups(); len(got) != 1 {
		t.Fatalf("KeyCompanyGroups len = %d", len(got))
	}
	if got := ind.TopPerformingCompanies(); len(got) != 1 {
		t.Fatalf("Industry.TopPerformingCompanies len = %d", len(got))
	}
}

func TestFundsDataTypedAccessors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"quoteSummary": map[string]any{
				"result": []any{map[string]any{
					"summaryProfile": map[string]any{
						"longBusinessSummary": "An ETF that tracks the technology sector.",
					},
					"fundProfile": map[string]any{
						"family":                  "Vanguard",
						"categoryName":            "Technology",
						"legalType":               "Exchange Traded Fund",
						"feesExpensesInvestment": map[string]any{"annualReportExpenseRatio": 0.001},
					},
					"topHoldings": map[string]any{
						"cashPosition":  0.01,
						"stockPosition": 0.99,
						"bondPosition":  0,
						"holdings": []any{
							map[string]any{"symbol": "AAPL", "holdingPercent": 0.18},
						},
						"equityHoldings":   map[string]any{"priceToBook": 8.0},
						"bondHoldings":     map[string]any{"duration": 0.0},
						"bondRatings":      []any{map[string]any{"a": 0.5}},
						"sectorWeightings": []any{map[string]any{"technology": 0.99}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	fd, err := client.FundsData(context.Background(), "VGT")
	if err != nil {
		t.Fatalf("FundsData: %v", err)
	}
	if fd.Symbol != "VGT" {
		t.Fatalf("Symbol = %q", fd.Symbol)
	}
	if !strings.Contains(fd.Description(), "technology sector") {
		t.Fatalf("Description = %q", fd.Description())
	}
	if ov := fd.FundOverview(); ov == nil || ov["family"] != "Vanguard" {
		t.Fatalf("FundOverview = %+v", ov)
	}
	if ops := fd.FundOperations(); ops == nil {
		t.Fatalf("FundOperations was nil")
	}
	if ac := fd.AssetClasses(); ac == nil || ac["stockPosition"] == nil {
		t.Fatalf("AssetClasses = %+v", ac)
	}
	if h := fd.TopHoldings(); len(h) != 1 {
		t.Fatalf("TopHoldings len = %d", len(h))
	}
	if fd.EquityHoldings() == nil {
		t.Fatalf("EquityHoldings nil")
	}
	if fd.BondHoldings() == nil {
		t.Fatalf("BondHoldings nil")
	}
	if len(fd.BondRatings()) != 1 || len(fd.SectorWeightings()) != 1 {
		t.Fatalf("BondRatings/SectorWeightings empty")
	}
}

func TestSearchRichAccessors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"quotes": []any{
				map[string]any{"symbol": "AAPL", "shortname": "Apple Inc."},
			},
			"news": []any{
				map[string]any{"title": "Apple earnings"},
			},
			"lists": []any{
				map[string]any{"slug": "most-active"},
			},
			"researchReports": []any{
				map[string]any{"title": "Apple research"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	resp, err := client.Search(context.Background(), "apple", 1, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got := resp.Lists(); len(got) != 1 || stringValue(got[0]["slug"]) != "most-active" {
		t.Fatalf("Lists = %+v", got)
	}
	if got := resp.Research(); len(got) != 1 {
		t.Fatalf("Research len = %d", len(got))
	}
	if got := resp.NewsRows(); len(got) != 1 {
		t.Fatalf("NewsRows len = %d", len(got))
	}
	all := resp.All()
	if all["quotes"] == nil || all["lists"] == nil || all["researchReports"] == nil {
		t.Fatalf("All() missing sections: %+v", all)
	}
}

func TestHistoryRepairFixesCentupleAnomaly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta": map[string]any{"symbol": "VOD.L"},
					"timestamp": []int64{
						1700000000, 1700086400, 1700172800, 1700259200,
						1700345600, 1700432000, 1700518400,
					},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open":   []any{1.05, 1.06, 105.0, 1.07, 1.08, 1.09, 1.10},
							"high":   []any{1.10, 1.11, 110.0, 1.12, 1.13, 1.14, 1.15},
							"low":    []any{1.00, 1.01, 100.0, 1.02, 1.03, 1.04, 1.05},
							"close":  []any{1.05, 1.06, 106.0, 1.07, 1.08, 1.09, 1.10},
							"volume": []any{1000, 1000, 1000, 1000, 1000, 1000, 1000},
						}},
						"adjclose": []any{map[string]any{"adjclose": []any{1.05, 1.06, 106.0, 1.07, 1.08, 1.09, 1.10}}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Ticker("VOD.L").History(context.Background(), HistoryParams{
		Period: Period5D,
		Repair: true,
	})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	bad := result.Candles[2]
	if bad.Close > 2 {
		t.Fatalf("repair did not normalize 100x close: %v", bad.Close)
	}
	if bad.Open > 2 || bad.High > 2 || bad.Low > 2 {
		t.Fatalf("repair did not normalize 100x OHL: %+v", bad)
	}
}

func TestDownloadHonorsThreadsLimit(t *testing.T) {
	var (
		mu      sync.Mutex
		current int
		peak    int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		current++
		if current > peak {
			peak = current
		}
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		current--
		mu.Unlock()
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	results := client.Download(context.Background(),
		[]string{"A", "B", "C", "D", "E", "F", "G", "H"},
		HistoryParams{Period: Period5D, Threads: 2})
	if len(results) != 8 {
		t.Fatalf("got %d results", len(results))
	}
	if peak > 2 {
		t.Fatalf("peak in-flight = %d, want <= 2", peak)
	}
}

func TestSharesQuarterlyTimeseries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("type"); !strings.Contains(got, "quarterlyShareIssued") {
			t.Fatalf("type = %q", got)
		}
		writeJSON(t, w, map[string]any{"timeseries": map[string]any{"result": []any{}}})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query2URL = server.URL
	if _, err := client.Ticker("AAPL").Shares(context.Background(), "quarterly"); err != nil {
		t.Fatalf("Shares: %v", err)
	}
}

func TestAuthenticateSetsCrumbFromGetCrumb(t *testing.T) {
	var (
		homeHits  int
		crumbHits int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			homeHits++
			http.SetCookie(w, &http.Cookie{Name: "A1", Value: "test"})
			w.Write([]byte("ok"))
		case "/v1/test/getcrumb":
			crumbHits++
			w.Write([]byte("FAKE-CRUMB"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(nil)
	client.RootURL = server.URL
	client.Query2URL = server.URL
	client.HTTPClient = server.Client()
	if client.HTTPClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		client.HTTPClient.Jar = jar
	}

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if client.Crumb != "FAKE-CRUMB" {
		t.Fatalf("Crumb = %q", client.Crumb)
	}
	if homeHits == 0 || crumbHits == 0 {
		t.Fatalf("expected both home and crumb hits, got %d/%d", homeHits, crumbHits)
	}
}

func TestRetryOnTransient5xx(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	client.Retries = 3
	client.RetryBackoff = time.Millisecond
	if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D}); err != nil {
		t.Fatalf("History after retry: %v", err)
	}
	if hits != 3 {
		t.Fatalf("expected 3 attempts, got %d", hits)
	}
}

func TestRetryGivesUpOnNonRetryable(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	client.Retries = 5
	client.RetryBackoff = time.Microsecond
	if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D}); err == nil {
		t.Fatalf("expected error for 400")
	}
	if hits != 1 {
		t.Fatalf("expected exactly 1 attempt for 400, got %d", hits)
	}
}

func TestLoggerEmitsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	client.Logger = logger
	if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D}); err != nil {
		t.Fatalf("History: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "yfinance request") || !strings.Contains(out, "yfinance response") {
		t.Fatalf("logger output missing request/response: %s", out)
	}
}

func TestRateLimiterGatesRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	client.Limiter = NewRateLimiter(20, 1)

	start := time.Now()
	for i := 0; i < 4; i++ {
		if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D}); err != nil {
			t.Fatalf("History: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 100*time.Millisecond {
		t.Fatalf("rate limiter did not gate; elapsed = %v", elapsed)
	}
}

func TestHistoryNoEventsSendsEmptyParam(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("events")
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D, NoEvents: true}); err != nil {
		t.Fatalf("History: %v", err)
	}
	if captured != "" {
		t.Fatalf("events = %q, want empty", captured)
	}
}

func TestDownloadOnProgressCalledForEachSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL

	var (
		mu     sync.Mutex
		seen   []string
		errors []error
	)
	results := client.Download(context.Background(),
		[]string{"A", "B", "C"},
		HistoryParams{Period: Period5D, OnProgress: func(sym string, idx, total int, err error) {
			mu.Lock()
			seen = append(seen, sym)
			errors = append(errors, err)
			mu.Unlock()
			if total != 3 {
				t.Errorf("total = %d, want 3", total)
			}
		}})
	if len(results) != 3 {
		t.Fatalf("got %d results", len(results))
	}
	if len(seen) != 3 {
		t.Fatalf("OnProgress called %d times, want 3", len(seen))
	}
	for _, e := range errors {
		if e != nil {
			t.Fatalf("unexpected progress error: %v", e)
		}
	}
}

func TestHistoryDropNaNFiltersAllNaNRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta":      map[string]any{"symbol": "X"},
					"timestamp": []int64{1, 2, 3},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open":   []any{1.0, nil, 3.0},
							"high":   []any{1.0, nil, 3.0},
							"low":    []any{1.0, nil, 3.0},
							"close":  []any{1.0, nil, 3.0},
							"volume": []any{10, 0, 30},
						}},
						"adjclose": []any{map[string]any{"adjclose": []any{1.0, nil, 3.0}}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Ticker("X").History(context.Background(), HistoryParams{
		Period:  Period5D,
		DropNaN: true,
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(result.Candles) != 2 {
		t.Fatalf("expected 2 rows after DropNaN, got %d", len(result.Candles))
	}
}

func TestRepairNaNsZeroOHLCWithHealthyNeighbors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"chart": map[string]any{
				"result": []any{map[string]any{
					"meta": map[string]any{"symbol": "X"},
					"timestamp": []int64{
						1, 2, 3, 4, 5, 6, 7,
					},
					"indicators": map[string]any{
						"quote": []any{map[string]any{
							"open":   []any{1.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0},
							"high":   []any{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
							"low":    []any{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
							"close":  []any{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
							"volume": []any{10, 10, 10, 10, 10, 10, 10},
						}},
						"adjclose": []any{map[string]any{"adjclose": []any{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}}},
					},
				}},
				"error": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	result, err := client.Ticker("X").History(context.Background(), HistoryParams{
		Period: Period5D,
		Repair: true,
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	bad := result.Candles[2]
	if !math.IsNaN(bad.Open) {
		t.Fatalf("expected zero Open to be NaN'd, got %v", bad.Open)
	}
}

func TestSearchWithOptionsForwardsFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query()
		if got.Get("enableFuzzyQuery") != "true" {
			t.Fatalf("enableFuzzyQuery = %q", got.Get("enableFuzzyQuery"))
		}
		if got.Get("recommendCount") != "5" {
			t.Fatalf("recommendCount = %q", got.Get("recommendCount"))
		}
		if got.Get("region") != "GB" {
			t.Fatalf("region = %q", got.Get("region"))
		}
		writeJSON(t, w, map[string]any{"quotes": []any{}, "news": []any{}})
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	yes := true
	if _, err := client.SearchWithOptions(context.Background(), "apple", SearchOptions{
		QuotesCount:      5,
		EnableFuzzyQuery: &yes,
		RecommendCount:   5,
		Region:           "GB",
	}); err != nil {
		t.Fatalf("SearchWithOptions: %v", err)
	}
}

func TestMemoryCacheSavesAndServesGetJSON(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		writeEmptyChart(t, w)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	client.Cache = NewMemoryCache()
	client.CacheTTL = time.Minute

	for i := 0; i < 3; i++ {
		if _, err := client.Ticker("A").History(context.Background(), HistoryParams{Period: Period5D}); err != nil {
			t.Fatalf("History: %v", err)
		}
	}
	if hits != 1 {
		t.Fatalf("server hits = %d, want 1 with cache", hits)
	}
}

func TestClientLimitsAndPostErrorMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.Query1URL = server.URL
	if _, err := client.Lookup(context.Background(), "apple", LookupAll, maxLookupCount+1); err == nil {
		t.Fatalf("expected lookup limit error")
	}
	if _, err := client.News(context.Background(), "aapl", maxNewsCount+1, "news"); err == nil {
		t.Fatalf("expected news limit error")
	}
	if _, err := client.FundamentalsTimeseries(context.Background(), "aapl", make([]string, maxTimeseriesTypes+1), time.Time{}, time.Time{}); err == nil {
		t.Fatalf("expected timeseries limit error")
	}
	_, err := client.Screen(context.Background(), ScreenRequest{Size: 1})
	if err == nil || !strings.Contains(err.Error(), "POST /v1/finance/screener") {
		t.Fatalf("Screen error = %v", err)
	}
}
