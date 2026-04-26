package yfinance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
