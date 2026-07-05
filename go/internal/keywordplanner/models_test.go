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
	_, _ = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "")

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
	_, _ = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "")

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
		_ = json.NewEncoder(w).Encode(map[string]any{"adGroupForecastMetrics": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	_, err := client.GetKeywordForecast(context.Background(), []string{"go"}, 0, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to parse captured request body: %v", err)
	}
	bidMicros := req["campaignForecastSpec"].(map[string]any)["biddingStrategy"].(map[string]any)["manualCpcBiddingStrategy"].(map[string]any)["maxCpcBidMicros"]
	if bidMicros != "1000000" {
		t.Errorf("maxCpcBidMicros = %v, want %q", bidMicros, "1000000")
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
	_, err := client.GenerateKeywordIdeas(context.Background(), []string{"test"}, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errStr := err.Error()
	// The full 500-byte body must be present — no truncation.
	if len(errStr) < 500 {
		t.Errorf("error message appears truncated: len=%d, want >= 500", len(errStr))
	}
}
