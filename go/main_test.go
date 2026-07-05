package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// TestNewServer_RegistersTools verifies that newServer builds a server with all
// three tools registered and listable via a real client session, catching invalid
// struct tags or schema-generation failures at test time rather than at runtime.
func TestNewServer_RegistersTools(t *testing.T) {
	t.Parallel()

	client := keywordplanner.NewClient("token", "id", "secret", "refresh", "123", "")
	srv := newServer(client)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	var names []string
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"generate_keyword_ideas", "get_historical_metrics", "get_keyword_forecast"} {
		if !slices.Contains(names, want) {
			t.Errorf("tool %q not registered; got tools %v", want, names)
		}
	}
}

// TestGenerateKeywordIdeasInput_LanguageField_HasDescription confirms that the
// language parameter's schema description tells callers to use the
// languageConstants/{id} format, so LLMs don't guess "en" or "1000".
func TestGenerateKeywordIdeasInput_LanguageField_HasDescription(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[generateKeywordIdeasInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	prop, ok := schema.Properties["language"]
	if !ok {
		t.Fatal("language property not found in schema")
	}

	if !strings.Contains(prop.Description, "languageConstants/") {
		t.Errorf("language description %q must contain 'languageConstants/' to guide callers", prop.Description)
	}
}

// TestGenerateKeywordIdeasInput_AllFields_HaveDescriptions confirms every field
// on generateKeywordIdeasInput carries a non-empty description.
func TestGenerateKeywordIdeasInput_AllFields_HaveDescriptions(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[generateKeywordIdeasInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	for name, prop := range schema.Properties {
		if prop.Description == "" {
			t.Errorf("field %q has no description", name)
		}
	}
}

// TestGetHistoricalMetricsInput_AllFields_HaveDescriptions confirms every field
// on getHistoricalMetricsInput carries a non-empty description.
func TestGetHistoricalMetricsInput_AllFields_HaveDescriptions(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[getHistoricalMetricsInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	for name, prop := range schema.Properties {
		if prop.Description == "" {
			t.Errorf("field %q has no description", name)
		}
	}
}

// TestGetKeywordForecastInput_AllFields_HaveDescriptions confirms every field
// on getKeywordForecastInput carries a non-empty description.
func TestGetKeywordForecastInput_AllFields_HaveDescriptions(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[getKeywordForecastInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	for name, prop := range schema.Properties {
		if prop.Description == "" {
			t.Errorf("field %q has no description", name)
		}
	}
}

// TestGenerateKeywordIdeasInput_SeedKeywordsURLLanguage_AreNotRequired confirms
// seed_keywords, url, and language are absent from the schema's required list.
// The tool description states "at least one of seed_keywords or url must be
// provided," which is not expressible as JSON Schema "required" (that would force
// both fields on every call); the constraint is instead enforced at runtime by
// generateKeywordIdeas.
func TestGenerateKeywordIdeasInput_SeedKeywordsURLLanguage_AreNotRequired(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[generateKeywordIdeasInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	for _, name := range []string{"seed_keywords", "url", "language"} {
		if slices.Contains(schema.Required, name) {
			t.Errorf("field %q must not be in schema.Required (got %v)", name, schema.Required)
		}
	}
}

// TestGetKeywordForecastInput_ForecastDays_IsNotRequired confirms forecast_days is
// absent from the schema's required list, matching its description ("Defaults to
// 30 if omitted or 0") and the defaulting behavior in Client.GetKeywordForecast.
func TestGetKeywordForecastInput_ForecastDays_IsNotRequired(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[getKeywordForecastInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	if slices.Contains(schema.Required, "forecast_days") {
		t.Errorf("forecast_days must not be in schema.Required (got %v)", schema.Required)
	}
}

// TestGetKeywordForecastInput_MaxCPCMicros_IsNotRequired confirms max_cpc_micros is
// absent from the schema's required list, matching its description ("Defaults to
// 1,000,000 if omitted or 0") and the defaulting behavior in Client.GetKeywordForecast.
func TestGetKeywordForecastInput_MaxCPCMicros_IsNotRequired(t *testing.T) {
	t.Parallel()

	schema, err := jsonschema.For[getKeywordForecastInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}

	if slices.Contains(schema.Required, "max_cpc_micros") {
		t.Errorf("max_cpc_micros must not be in schema.Required (got %v)", schema.Required)
	}
}

// TestGenerateKeywordIdeas_NoSeedKeywordsOrURL_ReturnsValidationError verifies the
// handler rejects a call providing neither seed_keywords nor url with a clear tool
// error, instead of forwarding an empty seed request to the Google Ads API.
func TestGenerateKeywordIdeas_NoSeedKeywordsOrURL_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	var apiHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiHit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := generateKeywordIdeas(context.Background(), client, generateKeywordIdeasInput{})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if apiHit {
		t.Error("Google Ads API must not be called when neither seed_keywords nor url is provided")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "at least one of seed_keywords or url must be provided") {
		t.Errorf("result text = %q, want it to mention the seed_keywords/url requirement", text)
	}
}

// TestGenerateKeywordIdeas_URLOnly_SkipsValidationError verifies that providing
// only a url (no seed_keywords) is accepted, since either input alone satisfies
// the tool's "at least one of" contract.
func TestGenerateKeywordIdeas_URLOnly_SkipsValidationError(t *testing.T) {
	t.Parallel()

	var apiHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiHit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	_, _, err := generateKeywordIdeas(context.Background(), client, generateKeywordIdeasInput{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !apiHit {
		t.Error("Google Ads API should be called when url is provided even without seed_keywords")
	}
}

// TestGenerateKeywordIdeas_APIError_ReturnsErrorContent is a characterization
// test written ahead of the go-sdk dependency upgrade (issue #10): it pins down
// the one previously-untested branch of generateKeywordIdeas, confirming a
// Google Ads API failure surfaces as a tool error result (JSON {"error": ...}
// text content) rather than a Go/protocol-level error.
func TestGenerateKeywordIdeas_APIError_ReturnsErrorContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "boom"}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := generateKeywordIdeas(context.Background(), client, generateKeywordIdeasInput{SeedKeywords: []string{"x"}})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "generating keyword ideas:") {
		t.Errorf("result text = %q, want it to mention %q", text, "generating keyword ideas:")
	}
}

// TestGetHistoricalMetrics_Success_ReturnsMarshaledMetrics is a characterization
// test written ahead of the go-sdk dependency upgrade (issue #10). Before this
// test, getHistoricalMetrics (the get_historical_metrics tool handler) had 0%
// coverage: nothing exercised its mcp.CallToolResult/mcp.TextContent
// construction, so a migration-induced regression here would have gone
// completely unnoticed.
func TestGetHistoricalMetrics_Success_ReturnsMarshaledMetrics(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"metrics": [{
				"text": "dependency injection",
				"keywordMetrics": {
					"avgMonthlySearches": "1000",
					"competition": "MEDIUM",
					"competitionIndex": 50,
					"lowTopOfPageBidMicros": "100000",
					"highTopOfPageBidMicros": "500000",
					"monthlySearchVolumes": []
				}
			}]
		}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := getHistoricalMetrics(context.Background(), client, getHistoricalMetricsInput{Keywords: []string{"dependency injection"}})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var parsed keywordplanner.HistoricalMetricsResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result content: %v", err)
	}
	if parsed.Count != 1 {
		t.Errorf("Count = %d, want 1", parsed.Count)
	}
	if len(parsed.Keywords) != 1 || parsed.Keywords[0].Text != "dependency injection" {
		t.Errorf("Keywords = %+v, want one entry for %q", parsed.Keywords, "dependency injection")
	}
}

// TestGetHistoricalMetrics_APIError_ReturnsErrorContent verifies a Google Ads
// API failure surfaces as a tool error result, not a Go/protocol-level error --
// mirroring the same already-proven pattern for generate_keyword_ideas.
func TestGetHistoricalMetrics_APIError_ReturnsErrorContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "boom"}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := getHistoricalMetrics(context.Background(), client, getHistoricalMetricsInput{Keywords: []string{"x"}})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "getting historical metrics:") {
		t.Errorf("result text = %q, want it to mention %q", text, "getting historical metrics:")
	}
}

// TestGetKeywordForecast_Success_ReturnsMarshaledForecast is a characterization
// test written ahead of the go-sdk dependency upgrade (issue #10). Before this
// test, getKeywordForecast (the get_keyword_forecast tool handler) had 0%
// coverage, for the same reason as getHistoricalMetrics above. It also
// confirms the default-forecast-window/default-CPC behavior survives the
// handler's JSON round-trip, not just the internal client call.
func TestGetKeywordForecast_Success_ReturnsMarshaledForecast(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"adGroupForecastMetrics": [{
				"keywordForecastMetrics": [{
					"keyword": {"text": "dependency injection", "matchType": "BROAD"},
					"metrics": {"impressions": 1000, "clicks": 50, "costMicros": 500000, "ctr": 0.05}
				}]
			}]
		}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := getKeywordForecast(context.Background(), client, getKeywordForecastInput{Keywords: []string{"dependency injection"}})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var parsed keywordplanner.ForecastResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result content: %v", err)
	}
	if len(parsed.Keywords) != 1 || parsed.Keywords[0].Text != "dependency injection" {
		t.Errorf("Keywords = %+v, want one entry for %q", parsed.Keywords, "dependency injection")
	}
	if parsed.ForecastDays != 30 {
		t.Errorf("ForecastDays = %d, want default of 30", parsed.ForecastDays)
	}
	if parsed.MaxCPCMicros != 1_000_000 {
		t.Errorf("MaxCPCMicros = %d, want default of 1,000,000", parsed.MaxCPCMicros)
	}
}

// TestGetKeywordForecast_APIError_ReturnsErrorContent verifies a Google Ads API
// failure surfaces as a tool error result, not a Go/protocol-level error.
func TestGetKeywordForecast_APIError_ReturnsErrorContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "boom"}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	result, _, err := getKeywordForecast(context.Background(), client, getKeywordForecastInput{Keywords: []string{"x"}})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "getting keyword forecast:") {
		t.Errorf("result text = %q, want it to mention %q", text, "getting keyword forecast:")
	}
}

// TestNewServer_CallHistoricalMetricsTool_ViaRealSession confirms the
// get_historical_metrics tool, as actually registered by newServer (not just
// the underlying Go function called directly), works end-to-end through a
// real MCP client session. Before this test, the closure newServer registers
// for this tool was never invoked by anything: argument binding, schema
// validation, and tool dispatch all have to agree for this to pass.
func TestNewServer_CallHistoricalMetricsTool_ViaRealSession(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metrics": []}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	mcpServer := newServer(client)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_historical_metrics",
		Arguments: map[string]any{"keywords": []string{"dependency injection"}},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Errorf("CallTool returned an error result: %+v", result.Content)
	}
}

// TestNewServer_CallKeywordForecastTool_ViaRealSession is the
// get_keyword_forecast equivalent of
// TestNewServer_CallHistoricalMetricsTool_ViaRealSession.
func TestNewServer_CallKeywordForecastTool_ViaRealSession(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"adGroupForecastMetrics": []}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	mcpServer := newServer(client)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_keyword_forecast",
		Arguments: map[string]any{"keywords": []string{"dependency injection"}},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Errorf("CallTool returned an error result: %+v", result.Content)
	}
}
