package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// TestNewServer_RegistersTools verifies that the MCP server can be created and all
// tools can be registered without panicking. This catches invalid struct tags or
// schema-generation failures at test time rather than at runtime.
func TestNewServer_RegistersTools(_ *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "google-keyword-planner-mcp",
		Version: "test",
	}, nil)

	client := keywordplanner.NewClient("token", "id", "secret", "refresh", "123", "")

	mcp.AddTool(srv,
		&mcp.Tool{Name: "generate_keyword_ideas", Description: "test"},
		func(ctx context.Context, _ *mcp.CallToolRequest, input generateKeywordIdeasInput) (*mcp.CallToolResult, any, error) {
			return generateKeywordIdeas(ctx, client, input)
		},
	)

	mcp.AddTool(srv,
		&mcp.Tool{Name: "get_historical_metrics", Description: "test"},
		func(ctx context.Context, _ *mcp.CallToolRequest, input getHistoricalMetricsInput) (*mcp.CallToolResult, any, error) {
			return getHistoricalMetrics(ctx, client, input)
		},
	)

	mcp.AddTool(srv,
		&mcp.Tool{Name: "get_keyword_forecast", Description: "test"},
		func(ctx context.Context, _ *mcp.CallToolRequest, input getKeywordForecastInput) (*mcp.CallToolResult, any, error) {
			return getKeywordForecast(ctx, client, input)
		},
	)
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
