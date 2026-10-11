package keywordplanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// lookupServer answers the GAQL lookups and keyword idea requests, recording
// the last keyword idea request body and how many lookups were made.
func lookupServer(t *testing.T, ideaBody *map[string]any, lookups *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/googleAds:search") {
			_ = json.NewDecoder(r.Body).Decode(ideaBody)
			_, _ = w.Write([]byte(`{"results": []}`))
			return
		}
		lookups.Add(1)
		var req struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if strings.Contains(req.Query, "FROM language_constant") {
			_, _ = w.Write([]byte(`{"results": [{"languageConstant": {"id": "1001", "code": "de", "name": "German"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"results": [{"geoTargetConstant": {"id": "2040", "countryCode": "AT", "name": "Austria"}}]}`))
	}))
}

func TestGenerateKeywordIdeas_ResolvesLanguageAndCountryCodes(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var lookups atomic.Int32
	srv := lookupServer(t, &body, &lookups)
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	for range 2 {
		_, err := client.GenerateKeywordIdeas(
			context.Background(), []string{"go"}, "", "DE", []string{"at", "geoTargetConstants/2276", "2756"}, "",
			keywordplanner.IdeaOptions{},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if body["language"] != "languageConstants/1001" {
		t.Errorf("language = %v, want languageConstants/1001", body["language"])
	}
	geo, _ := body["geoTargetConstants"].([]any)
	want := []string{"geoTargetConstants/2040", "geoTargetConstants/2276", "geoTargetConstants/2756"}
	if len(geo) != len(want) {
		t.Fatalf("geoTargetConstants = %v, want %v", geo, want)
	}
	for i, w := range want {
		if geo[i] != w {
			t.Errorf("geoTargetConstants[%d] = %v, want %v", i, geo[i], w)
		}
	}
	// One language table and one country table, fetched once and then cached.
	if got := lookups.Load(); got != 2 {
		t.Errorf("lookups = %d, want 2", got)
	}
}

func TestGenerateKeywordIdeas_DefaultTargetingAcceptsCodes(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var lookups atomic.Int32
	srv := lookupServer(t, &body, &lookups)
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	client.SetDefaultTargeting("de", []string{"AT"})
	if _, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["language"] != "languageConstants/1001" {
		t.Errorf("language = %v, want languageConstants/1001", body["language"])
	}
	geo, _ := body["geoTargetConstants"].([]any)
	if len(geo) != 1 || geo[0] != "geoTargetConstants/2040" {
		t.Errorf("geoTargetConstants = %v, want [geoTargetConstants/2040]", geo)
	}
}

func TestGenerateKeywordIdeas_UnknownGeoTargetIsAnError(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var lookups atomic.Int32
	srv := lookupServer(t, &body, &lookups)
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", []string{"Atlantis"}, "", keywordplanner.IdeaOptions{})
	if err == nil || !strings.Contains(err.Error(), "search_geo_targets") {
		t.Fatalf("err = %v, want an error pointing at search_geo_targets", err)
	}
	if body != nil {
		t.Errorf("keyword ideas were requested despite the unknown geo target: %v", body)
	}
}

func TestGenerateKeywordIdeas_SortsFiltersAndLimits(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": [
			{"text": "small", "keywordIdeaMetrics": {"avgMonthlySearches": "10"}},
			{"text": "big", "keywordIdeaMetrics": {"avgMonthlySearches": "900", "competitionIndex": "78",
				"monthlySearchVolumes": [{"year": "2026", "month": "AUGUST", "monthlySearches": "1000"}]}},
			{"text": "mid", "keywordIdeaMetrics": {"avgMonthlySearches": "200"}},
			{"text": "upper", "keywordIdeaMetrics": {"avgMonthlySearches": "500"}}
		]}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "",
		keywordplanner.IdeaOptions{Limit: 2, MinMonthlySearches: 100, IncludeMonthlyVolumes: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalFound != 4 || result.Count != 2 {
		t.Fatalf("TotalFound = %d, Count = %d, want 4 and 2", result.TotalFound, result.Count)
	}
	if result.Ideas[0].Text != "big" || result.Ideas[1].Text != "upper" {
		t.Errorf("ideas = %q, %q, want big, upper", result.Ideas[0].Text, result.Ideas[1].Text)
	}
	big := result.Ideas[0]
	if big.CompetitionIndex != 78 {
		t.Errorf("CompetitionIndex = %d, want 78", big.CompetitionIndex)
	}
	if len(big.MonthlySearchVolumes) != 1 || big.MonthlySearchVolumes[0].Year != 2026 ||
		big.MonthlySearchVolumes[0].Month != 8 || big.MonthlySearchVolumes[0].MonthlySearches != 1000 {
		t.Errorf("MonthlySearchVolumes = %+v, want one entry for 2026-08 with 1000", big.MonthlySearchVolumes)
	}
}

func TestGenerateKeywordIdeas_OmitsMonthlyVolumesByDefault(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": [{"text": "big", "keywordIdeaMetrics": {"avgMonthlySearches": "900",
			"monthlySearchVolumes": [{"year": "2026", "month": "AUGUST", "monthlySearches": "1000"}]}}]}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Ideas) != 1 || result.Ideas[0].MonthlySearchVolumes != nil {
		t.Errorf("ideas = %+v, want one idea without monthly volumes", result.Ideas)
	}
}

func TestSearchGeoTargets_MapsSuggestions(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/geoTargetConstants:suggest") {
			t.Errorf("path = %s, want .../geoTargetConstants:suggest", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"geoTargetConstantSuggestions": [{"geoTargetConstant": {
			"resourceName": "geoTargetConstants/20050", "name": "Wien", "countryCode": "AT",
			"targetType": "State", "canonicalName": "Vienna,Austria"}, "reach": "7010000"}]}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.SearchGeoTargets(context.Background(), []string{"Wien"}, "at", "de")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["countryCode"] != "AT" || body["locale"] != "de" {
		t.Errorf("request = %v, want countryCode AT and locale de", body)
	}
	if result.Count != 1 {
		t.Fatalf("Count = %d, want 1", result.Count)
	}
	got := result.GeoTargets[0]
	if got.ResourceName != "geoTargetConstants/20050" || got.TargetType != "State" || got.Reach != 7010000 {
		t.Errorf("geo target = %+v, want Vienna state with reach 7010000", got)
	}
}
