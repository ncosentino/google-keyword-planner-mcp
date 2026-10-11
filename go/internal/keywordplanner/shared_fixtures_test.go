package keywordplanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func sharedFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "google-ads", "v25", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestSharedHistoricalFixture(t *testing.T) {
	body := sharedFixture(t, "historical.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := keywordplanner.NewTestClient("token", "123", "", server.URL, server.Client())
	result, err := client.GetHistoricalMetrics(context.Background(), []string{"dependency injection"}, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Fatalf("count = %d", result.Count)
	}
	keyword := result.Keywords[0]
	if keyword.AvgMonthlySearches != 5000 || keyword.CompetitionIndex != 23 ||
		len(keyword.MonthlySearchVolumes) != 1 || keyword.MonthlySearchVolumes[0].Year != 2026 ||
		keyword.MonthlySearchVolumes[0].Month != 9 || keyword.MonthlySearchVolumes[0].MonthlySearches != 4500 {
		t.Fatalf("metrics = %+v", keyword)
	}
}

func TestSharedForecastFixturesPreserveOptionalMetrics(t *testing.T) {
	for _, test := range []struct {
		name string
		present bool
		clicks float64
		cost int64
	}{
		{"forecast.json", true, 12.5, 9000000000},
		{"forecast-zero.json", true, 0, 0},
		{"forecast-unavailable.json", false, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := sharedFixture(t, test.name)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(body)
			}))
			defer server.Close()
			client := keywordplanner.NewTestClient("token", "123", "", server.URL, server.Client())
			result, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{Keywords: []string{"go"}})
			if err != nil {
				t.Fatal(err)
			}
			if (result.Total.Clicks != nil) != test.present || (result.Total.CostMicros != nil) != test.present {
				t.Fatalf("metric presence = %+v", result.Total)
			}
			if test.present && (*result.Total.Clicks != test.clicks || *result.Total.CostMicros != test.cost) {
				t.Fatalf("metrics = %+v", result.Total)
			}
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]json.RawMessage
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if _, ok := wire["total"]; !ok {
				t.Fatal("missing campaign total")
			}
		})
	}
}
