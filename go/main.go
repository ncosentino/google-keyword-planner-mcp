// Command google-keyword-planner-mcp is an MCP server that exposes Google Ads Keyword Planner
// as tools for AI assistants. It communicates via STDIO using the MCP protocol.
//
// Usage:
//
//	google-keyword-planner-mcp [--developer-token <token>] [--client-id <id>]
//	    [--client-secret <secret>] [--refresh-token <token>] [--customer-id <id>]
//
// Credential resolution order: CLI flags > environment variables > .env file.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/config"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

var version = "dev"

func main() {
	developerToken := flag.String("developer-token", "", "Google Ads developer token")
	clientID := flag.String("client-id", "", "OAuth2 client ID")
	clientSecret := flag.String("client-secret", "", "OAuth2 client secret")
	refreshToken := flag.String("refresh-token", "", "OAuth2 refresh token")
	customerID := flag.String("customer-id", "", "Google Ads customer ID")
	flag.Parse()

	// All diagnostic output must go to stderr to avoid corrupting the MCP STDIO stream.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Resolve(config.Flags{
		DeveloperToken: *developerToken,
		ClientID:       *clientID,
		ClientSecret:   *clientSecret,
		RefreshToken:   *refreshToken,
		CustomerID:     *customerID,
	})

	if !cfg.IsComplete() {
		slog.Error("incomplete Google Ads credentials",
			"hint", "set GOOGLE_ADS_DEVELOPER_TOKEN, GOOGLE_ADS_CLIENT_ID, "+
				"GOOGLE_ADS_CLIENT_SECRET, GOOGLE_ADS_REFRESH_TOKEN, GOOGLE_ADS_CUSTOMER_ID")
		os.Exit(1)
	}

	client := keywordplanner.NewClient(
		cfg.DeveloperToken, cfg.ClientID, cfg.ClientSecret, cfg.RefreshToken, cfg.CustomerID,
	)

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "google-keyword-planner-mcp",
		Version: version,
	}, nil)

	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "generate_keyword_ideas",
			Description: "Generate keyword ideas from seed keywords and/or a URL using Google Ads Keyword Planner. Returns related keywords with average monthly search volume, competition level, and CPC estimates.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input generateKeywordIdeasInput) (*mcp.CallToolResult, any, error) {
			return generateKeywordIdeas(ctx, client, input)
		},
	)

	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "get_historical_metrics",
			Description: "Get historical search volume and competition metrics for a list of specific keywords using Google Ads Keyword Planner.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input getHistoricalMetricsInput) (*mcp.CallToolResult, any, error) {
			return getHistoricalMetrics(ctx, client, input)
		},
	)

	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "get_keyword_forecast",
			Description: "Get projected impressions, clicks, and cost for a set of keywords at a given max CPC bid using Google Ads Keyword Planner.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input getKeywordForecastInput) (*mcp.CallToolResult, any, error) {
			return getKeywordForecast(ctx, client, input)
		},
	)

	slog.Info("google-keyword-planner-mcp starting", "version", version, "transport", "stdio")
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

// generateKeywordIdeasInput is the input schema for the generate_keyword_ideas tool.
type generateKeywordIdeasInput struct {
	SeedKeywords []string `json:"seed_keywords"`
	URL          string   `json:"url"`
	Language     string   `json:"language"`
}

// getHistoricalMetricsInput is the input schema for the get_historical_metrics tool.
type getHistoricalMetricsInput struct {
	Keywords []string `json:"keywords"`
}

// getKeywordForecastInput is the input schema for the get_keyword_forecast tool.
type getKeywordForecastInput struct {
	Keywords     []string `json:"keywords"`
	MaxCPCMicros int64    `json:"max_cpc_micros"`
	ForecastDays int      `json:"forecast_days"`
}

func generateKeywordIdeas(ctx context.Context, client *keywordplanner.Client, input generateKeywordIdeasInput) (*mcp.CallToolResult, any, error) {
	result, err := client.GenerateKeywordIdeas(ctx, input.SeedKeywords, input.URL, input.Language)
	if err != nil {
		errResult := map[string]string{"error": fmt.Sprintf("generating keyword ideas: %v", err)}
		b, _ := json.Marshal(errResult)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func getHistoricalMetrics(ctx context.Context, client *keywordplanner.Client, input getHistoricalMetricsInput) (*mcp.CallToolResult, any, error) {
	result, err := client.GetHistoricalMetrics(ctx, input.Keywords)
	if err != nil {
		errResult := map[string]string{"error": fmt.Sprintf("getting historical metrics: %v", err)}
		b, _ := json.Marshal(errResult)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func getKeywordForecast(ctx context.Context, client *keywordplanner.Client, input getKeywordForecastInput) (*mcp.CallToolResult, any, error) {
	result, err := client.GetKeywordForecast(ctx, input.Keywords, input.MaxCPCMicros, input.ForecastDays)
	if err != nil {
		errResult := map[string]string{"error": fmt.Sprintf("getting keyword forecast: %v", err)}
		b, _ := json.Marshal(errResult)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}
