package keywordplanner_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func TestNewClient_NotNil(t *testing.T) {
	t.Parallel()
	client := keywordplanner.NewClient("dev-token", "client-id", "client-secret", "refresh-token", "1234567890", "")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_WithLoginCustomerID_NotNil(t *testing.T) {
	t.Parallel()
	client := keywordplanner.NewClient("dev-token", "client-id", "client-secret", "refresh-token", "1234567890", "9876543210")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestKeywordIdeasResponse_Count(t *testing.T) {
	t.Parallel()
	resp := &keywordplanner.KeywordIdeasResponse{
		Ideas: []keywordplanner.KeywordIdea{
			{Text: "golang tutorial", AvgMonthlySearches: 5000, Competition: "LOW"},
			{Text: "go programming", AvgMonthlySearches: 8000, Competition: "MEDIUM"},
		},
		Count: 2,
	}
	if resp.Count != len(resp.Ideas) {
		t.Errorf("Count = %d, want %d", resp.Count, len(resp.Ideas))
	}
}

func TestHistoricalMetricsResponse_Count(t *testing.T) {
	t.Parallel()
	resp := &keywordplanner.HistoricalMetricsResponse{
		Keywords: []keywordplanner.KeywordMetrics{
			{Text: "blazor", AvgMonthlySearches: 12000, Competition: "LOW"},
		},
		Count: 1,
	}
	if resp.Count != len(resp.Keywords) {
		t.Errorf("Count = %d, want %d", resp.Count, len(resp.Keywords))
	}
}

// TestGenerateKeywordIdeas_SendsLoginCustomerIDHeader verifies the login-customer-id header
// is included when a manager account ID is configured.
func TestGenerateKeywordIdeas_SendsLoginCustomerIDHeader(t *testing.T) {
	t.Parallel()

	var capturedLoginID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLoginID = r.Header.Get("login-customer-id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient(
		"dev-token", "3778350596", "1381404200", srv.URL, srv.Client(),
	)
	_, _ = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{})

	if capturedLoginID != "1381404200" {
		t.Errorf("login-customer-id header = %q, want %q", capturedLoginID, "1381404200")
	}
}

// TestGenerateKeywordIdeas_OmitsLoginCustomerIDHeaderWhenEmpty verifies the header
// is not sent when no manager account ID is configured.
func TestGenerateKeywordIdeas_OmitsLoginCustomerIDHeaderWhenEmpty(t *testing.T) {
	t.Parallel()

	var capturedLoginID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLoginID = r.Header.Get("login-customer-id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient(
		"dev-token", "3778350596", "", srv.URL, srv.Client(),
	)
	_, _ = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{})

	if capturedLoginID != "" {
		t.Errorf("login-customer-id header should be absent, got %q", capturedLoginID)
	}
}

// TestPost_ReturnsFullErrorBody verifies that API error bodies are not truncated.
// TestGetKeywordForecast_ZeroMaxCPCMicros_DefaultsTo1Million verifies that a
// non-positive maxCPCMicros is replaced with the default bid of 1,000,000 micros
// ($1.00), matching the C# implementation's default and the schema's documented
// "Defaults to 1,000,000 if omitted or 0" behavior.
func TestGetKeywordForecast_ZeroMaxCPCMicros_DefaultsTo1Million(t *testing.T) {
	t.Parallel()

	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"campaignForecastMetrics": map[string]any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{
		Keywords: []string{"go"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to parse captured request body: %v", err)
	}
	bidMicros := req["campaign"].(map[string]any)["biddingStrategy"].(map[string]any)["manualCpcBiddingStrategy"].(map[string]any)["maxCpcBidMicros"]
	if bidMicros != "1000000" {
		t.Errorf("maxCpcBidMicros = %v, want %q", bidMicros, "1000000")
	}
}

// TestGetKeywordForecast_SendsV25RequestShape pins the request body to the
// GenerateKeywordForecastMetricsRequest shape: the forecast period is a
// top-level DateRange, the campaign is a CampaignToForecast, and the ad group
// carries a plain "keywords" list. The pre-v24 "campaignForecastSpec" wrapper
// and "biddableKeywords" list are rejected by the API with HTTP 400.
func TestGetKeywordForecast_SendsV25RequestShape(t *testing.T) {
	t.Parallel()

	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"campaignForecastMetrics": map[string]any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{
		Keywords:           []string{"kurs ceramiki"},
		MatchType:          "phrase",
		GeoTargetConstants: []string{"geoTargetConstants/2616"},
		Language:           "languageConstants/1030",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to parse captured request body: %v", err)
	}

	if _, ok := req["campaignForecastSpec"]; ok {
		t.Error("request still carries the removed campaignForecastSpec wrapper")
	}
	period, ok := req["forecastPeriod"].(map[string]any)
	if !ok {
		t.Fatal("request is missing the top-level forecastPeriod")
	}
	if period["startDate"] == "" || period["endDate"] == "" {
		t.Errorf("forecastPeriod is incomplete: %v", period)
	}

	campaign := req["campaign"].(map[string]any)
	if got := campaign["geoTargetConstants"].([]any)[0]; got != "geoTargetConstants/2616" {
		t.Errorf("geoTargetConstants[0] = %v, want geo target 2616", got)
	}
	if got := campaign["languageConstants"].([]any)[0]; got != "languageConstants/1030" {
		t.Errorf("languageConstants[0] = %v, want Polish", got)
	}

	adGroup := campaign["adGroups"].([]any)[0].(map[string]any)
	if _, ok := adGroup["biddableKeywords"]; ok {
		t.Error("ad group still carries the removed biddableKeywords list")
	}
	keyword := adGroup["keywords"].([]any)[0].(map[string]any)
	if keyword["text"] != "kurs ceramiki" {
		t.Errorf("keyword text = %v, want %q", keyword["text"], "kurs ceramiki")
	}
	if keyword["matchType"] != "PHRASE" {
		t.Errorf("matchType = %v, want PHRASE", keyword["matchType"])
	}
}

// TestGetKeywordForecast_ForecastsEachKeywordAndTheCombinedAdGroup verifies the
// per-keyword breakdown. The API returns a single campaign-level aggregate per
// request, so N keywords require N requests plus one for the combined total.
func TestGetKeywordForecast_ForecastsEachKeywordAndTheCombinedAdGroup(t *testing.T) {
	t.Parallel()

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"campaignForecastMetrics": map[string]any{
				"clicks":     10.5,
				"costMicros": "14285916",
			},
		})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{
		Keywords:                []string{"kurs ceramiki", "warsztaty ceramiczne"},
		IncludeKeywordBreakdown: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requests != 3 {
		t.Errorf("made %d requests, want 3 (one per keyword plus the combined total)", requests)
	}
	if len(result.Keywords) != 2 {
		t.Fatalf("got %d per-keyword forecasts, want 2", len(result.Keywords))
	}
	if result.Keywords[0].Text != "kurs ceramiki" {
		t.Errorf("Keywords[0].Text = %q, want %q", result.Keywords[0].Text, "kurs ceramiki")
	}
	if result.Keywords[0].CostMicros == nil || *result.Keywords[0].CostMicros != 14285916 {
		t.Errorf("Keywords[0].CostMicros = %v, want 14285916", result.Keywords[0].CostMicros)
	}
	if result.Total.Clicks == nil || *result.Total.Clicks != 10.5 {
		t.Errorf("Total.Clicks = %v, want 10.5", result.Total.Clicks)
	}
}

// TestGetHistoricalMetrics_ParsesResultsField pins the response field name.
// GenerateKeywordHistoricalMetricsResponse returns "results"; parsing "metrics"
// produced an empty keyword list on every call without surfacing an error.
func TestGetHistoricalMetrics_ParsesResultsField(t *testing.T) {
	t.Parallel()

	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []any{
				map[string]any{
					"text": "warsztaty ceramiczne",
					"keywordMetrics": map[string]any{
						"avgMonthlySearches": "2400",
						"competition":        "HIGH",
						"competitionIndex":   "67",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.GetHistoricalMetrics(
		context.Background(),
		[]string{"warsztaty ceramiczne"},
		"languageConstants/1030",
		[]string{"geoTargetConstants/2616"},
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Count = %d, want 1", result.Count)
	}
	if result.Keywords[0].AvgMonthlySearches != 2400 {
		t.Errorf("AvgMonthlySearches = %d, want 2400", result.Keywords[0].AvgMonthlySearches)
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to parse captured request body: %v", err)
	}
	if req["language"] != "languageConstants/1030" {
		t.Errorf("language = %v, want Polish", req["language"])
	}
	if got := req["geoTargetConstants"].([]any)[0]; got != "geoTargetConstants/2616" {
		t.Errorf("geoTargetConstants[0] = %v, want geo target 2616", got)
	}
}

// TestGenerateKeywordIdeas_SendsGeoTargetAndNetwork verifies the geo target and
// network reach the request body. Without a geo target the Ads API returns
// worldwide aggregate volumes, so silently dropping this field yields numbers
// that look plausible but describe the wrong country.
func TestGenerateKeywordIdeas_SendsGeoTargetAndNetwork(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, _ = client.GenerateKeywordIdeas(
		context.Background(), []string{"go"}, "", "languageConstants/1000",
		[]string{"geoTargetConstants/2036"}, "GOOGLE_SEARCH", keywordplanner.IdeaOptions{},
	)

	geo, ok := body["geoTargetConstants"].([]any)
	if !ok || len(geo) != 1 || geo[0] != "geoTargetConstants/2036" {
		t.Errorf("geoTargetConstants = %v, want [geoTargetConstants/2036]", body["geoTargetConstants"])
	}
	if body["keywordPlanNetwork"] != "GOOGLE_SEARCH" {
		t.Errorf("keywordPlanNetwork = %v, want GOOGLE_SEARCH", body["keywordPlanNetwork"])
	}
}

// TestGenerateKeywordIdeas_OmitsGeoTargetWhenEmpty verifies callers passing no
// geo target produce a request identical to the pre-geo behaviour.
func TestGenerateKeywordIdeas_OmitsGeoTargetWhenEmpty(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, _ = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{})

	if _, present := body["geoTargetConstants"]; present {
		t.Errorf("geoTargetConstants should be absent, got %v", body["geoTargetConstants"])
	}
	if _, present := body["keywordPlanNetwork"]; present {
		t.Errorf("keywordPlanNetwork should be absent, got %v", body["keywordPlanNetwork"])
	}
}

// TestGetHistoricalMetrics_SendsGeoTarget verifies geo targeting reaches the
// historical-metrics request too, which is the call most likely to be compared
// against locally-sourced volume figures.
func TestGetHistoricalMetrics_SendsGeoTarget(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, _ = client.GetHistoricalMetrics(
		context.Background(), []string{"go"},
		"languageConstants/1000", []string{"geoTargetConstants/2036"}, "GOOGLE_SEARCH",
	)

	geo, ok := body["geoTargetConstants"].([]any)
	if !ok || len(geo) != 1 || geo[0] != "geoTargetConstants/2036" {
		t.Errorf("geoTargetConstants = %v, want [geoTargetConstants/2036]", body["geoTargetConstants"])
	}
	if body["language"] != "languageConstants/1000" {
		t.Errorf("language = %v, want languageConstants/1000", body["language"])
	}
}

func TestPost_ReturnsFullErrorBody(t *testing.T) {
	t.Parallel()

	longBody := make([]byte, 500)
	for i := range longBody {
		longBody[i] = 'x'
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(longBody)
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, err := client.GenerateKeywordIdeas(context.Background(), []string{"test"}, "", "", nil, "", keywordplanner.IdeaOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errStr := err.Error()
	// The full 500-byte body must be present — no truncation.
	if len(errStr) < 500 {
		t.Errorf("error message appears truncated: len=%d, want >= 500", len(errStr))
	}
}
