package keywordplanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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

func TestSharedIdeaFixturePreservesZeroCompetition(t *testing.T) {
	body := sharedFixture(t, "ideas.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := keywordplanner.NewTestClient("token", "123", "", server.URL, server.Client())
	result, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "",
		keywordplanner.IdeaOptions{IncludeMonthlyVolumes: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.TotalFound != 1 || result.Ideas[0].AvgMonthlySearches != 5000 ||
		len(result.Ideas[0].MonthlySearchVolumes) != 1 || result.Ideas[0].MonthlySearchVolumes[0].MonthlySearches != 4500 {
		t.Fatalf("ideas = %+v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Ideas []map[string]json.RawMessage `json:"ideas"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if string(wire.Ideas[0]["competitionIndex"]) != "0" {
		t.Fatal("zero competition index must be present")
	}
}

func TestSharedForecastRequestFixture(t *testing.T) {
	var expected map[string]any
	if err := json.Unmarshal(sharedFixture(t, "forecast-request.json"), &expected); err != nil {
		t.Fatal(err)
	}
	body := sharedFixture(t, "forecast.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var actual map[string]any
		if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
			t.Error(err)
			return
		}
		period, ok := actual["forecastPeriod"].(map[string]any)
		if !ok {
			t.Error("missing forecastPeriod object")
			return
		}
		startText, ok := period["startDate"].(string)
		if !ok {
			t.Error("startDate is not a string")
			return
		}
		start, err := time.Parse("2006-01-02", startText)
		if err != nil || period["endDate"] != start.AddDate(0, 0, 29).Format("2006-01-02") {
			t.Errorf("unexpected inclusive date period: %v", period)
		}
		period["startDate"], period["endDate"] = "$start", "$end"
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("request = %#v, want %#v", actual, expected)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := keywordplanner.NewTestClient("token", "123", "", server.URL, server.Client())
	if _, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{
		Keywords: []string{"go"}, MatchType: "PHRASE", MaxCPCMicros: 2000000,
		ForecastDays: 30, Language: "languageConstants/1001",
		GeoTargetConstants: []string{"geoTargetConstants/2040"}, CurrencyCode: "EUR",
	}); err != nil {
		t.Fatal(err)
	}
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
		name    string
		present bool
		clicks  float64
		cost    int64
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
