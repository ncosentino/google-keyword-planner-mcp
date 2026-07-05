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
