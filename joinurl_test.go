package yfinance

import (
	"net/url"
	"testing"
)

// TestJoinURLPreservesPercentEscapesInPath is a regression test for the
// double-encoding bug that caused every Yahoo index ticker (^VIX, ^TNX,
// ^GSPC, ^DJI, ^IXIC, …) to return 404 "No data found, symbol may be
// delisted" since the early-2026 release. The pre-fix joinURL passed
// the result of `url.JoinPath` (which returns an already-encoded path)
// to `u.Path` (which expects a *decoded* path), so `u.String()`
// re-encoded the literal "%" → "%25" and Yahoo received e.g.
// "/v8/finance/chart/%255EVIX" instead of "/v8/finance/chart/%5EVIX".
//
// The fix is to use the URL method `(*url.URL).JoinPath` which sets
// both u.Path and u.RawPath correctly, so callers can keep passing
// pre-escaped path segments without paying double.
func TestJoinURLPreservesPercentEscapesInPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "caret-prefixed index symbol",
			path: "/v8/finance/chart/" + url.PathEscape("^VIX"),
			want: "https://query1.finance.yahoo.com/v8/finance/chart/%5EVIX",
		},
		{
			name: "raw caret without pre-escape works too",
			path: "/v8/finance/chart/^VIX",
			want: "https://query1.finance.yahoo.com/v8/finance/chart/%5EVIX",
		},
		{
			name: "berkshire class share",
			path: "/v8/finance/chart/" + url.PathEscape("BRK.B"),
			want: "https://query1.finance.yahoo.com/v8/finance/chart/BRK.B",
		},
		{
			name: "plain symbol",
			path: "/v8/finance/chart/SPY",
			want: "https://query1.finance.yahoo.com/v8/finance/chart/SPY",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := joinURL("https://query1.finance.yahoo.com", tc.path, nil)
			if err != nil {
				t.Fatalf("joinURL: %v", err)
			}
			if got != tc.want {
				t.Errorf("joinURL = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestJoinURLAppendsQueryParams(t *testing.T) {
	q := url.Values{}
	q.Set("range", "1mo")
	q.Set("interval", "1d")
	got, err := joinURL("https://query1.finance.yahoo.com", "/v8/finance/chart/"+url.PathEscape("^VIX"), q)
	if err != nil {
		t.Fatalf("joinURL: %v", err)
	}
	const want = "https://query1.finance.yahoo.com/v8/finance/chart/%5EVIX?interval=1d&range=1mo"
	if got != want {
		t.Errorf("joinURL = %q, want %q", got, want)
	}
}
