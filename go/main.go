// Command google-keyword-planner-mcp is an MCP server that exposes Google Ads Keyword Planner
// as tools for AI assistants. It supports STDIO transport (default) and HTTP transport.
//
// Usage:
//
//	google-keyword-planner-mcp [--transport stdio|http]
//	    [--listen-address <address>] [--port <port>] [--allowed-hosts <list>]
//	    [--developer-token <token>] [--client-id <id>] [--client-secret <secret>]
//	    [--refresh-token <token>] [--customer-id <id>]
//
// Credential resolution order: CLI flags > environment variables > .env file.
// When --transport http, MCP_LISTEN_ADDRESS and PORT configure the listener.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
	loginCustomerID := flag.String("login-customer-id", "", "Google Ads manager/MCC account ID (required when customer-id is a sub-account)")
	transport := flag.String("transport", "stdio", "Transport mode: stdio or http")
	listenAddress := flag.String(
		"listen-address",
		"",
		"HTTP listen address (default MCP_LISTEN_ADDRESS or 127.0.0.1)",
	)
	port := flag.Int("port", 0, "HTTP listen port (default PORT or 8080)")
	allowedHosts := flag.String("allowed-hosts", "localhost,127.0.0.1,[::1]",
		"Comma-separated Host header allow-list for --transport http (protects against DNS rebinding)")
	flag.Parse()
	explicitFlags := make(map[string]bool)
	flag.Visit(func(definedFlag *flag.Flag) {
		explicitFlags[definedFlag.Name] = true
	})

	// All diagnostic output must go to stderr to avoid corrupting the MCP STDIO stream.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Resolve(config.Flags{
		DeveloperToken:  *developerToken,
		ClientID:        *clientID,
		ClientSecret:    *clientSecret,
		RefreshToken:    *refreshToken,
		CustomerID:      *customerID,
		LoginCustomerID: *loginCustomerID,
	})

	if !cfg.IsComplete() {
		slog.Error("incomplete Google Ads credentials",
			"hint", "set GOOGLE_ADS_DEVELOPER_TOKEN, GOOGLE_ADS_CLIENT_ID, "+
				"GOOGLE_ADS_CLIENT_SECRET, GOOGLE_ADS_REFRESH_TOKEN, GOOGLE_ADS_CUSTOMER_ID")
		os.Exit(1)
	}

	client := keywordplanner.NewClient(
		cfg.DeveloperToken, cfg.ClientID, cfg.ClientSecret, cfg.RefreshToken, cfg.CustomerID, cfg.LoginCustomerID,
	)

	srv := newServer(client)

	switch *transport {
	case "http":
		httpListenAddress, err := resolveHTTPListenAddress(
			*listenAddress,
			explicitFlags["listen-address"],
		)
		if err != nil {
			slog.Error("invalid HTTP listen address", "err", err)
			os.Exit(1)
		}
		httpPort, err := resolveHTTPPort(*port, explicitFlags["port"])
		if err != nil {
			slog.Error("invalid HTTP port", "err", err)
			os.Exit(1)
		}
		ctx, stop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer stop()
		if err := runHTTP(ctx, srv, httpServerOptions{
			ListenAddress: httpListenAddress,
			Port:          httpPort,
			AllowedHosts:  splitAndTrim(*allowedHosts),
			ShutdownToken: strings.TrimSpace(os.Getenv("MCP_SHUTDOWN_TOKEN")),
		}); err != nil {
			slog.Error("server stopped with error", "err", err)
			os.Exit(1)
		}
	case "stdio":
		slog.Info("google-keyword-planner-mcp starting", "version", version, "transport", "stdio")
		if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("server stopped with error", "err", err)
			os.Exit(1)
		}
	default:
		slog.Error("invalid transport", "transport", *transport, "expected", "stdio or http")
		os.Exit(1)
	}
}

// newServer builds the MCP server with all tools and middleware registered. It is
// independent of which transport (stdio or http) will ultimately serve it.
func newServer(client *keywordplanner.Client) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "google-keyword-planner-mcp",
		Version: version,
	}, nil)

	// Repair a widespread MCP client bug where array-typed arguments arrive
	// JSON-encoded as a string instead of a genuine array (see stringified_args.go).
	srv.AddReceivingMiddleware(coerceStringifiedArrayArgs(toolArrayFields))

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

	return srv
}

// splitAndTrim splits a comma-separated flag value into a trimmed, non-empty slice.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// generateKeywordIdeasInput is the input schema for the generate_keyword_ideas tool.
// SeedKeywords and URL both carry ",omitempty" so neither is marked required in the
// exported JSON schema: the tool only requires that at least one of them be provided,
// which is enforced at runtime in generateKeywordIdeas rather than by the schema.
type generateKeywordIdeasInput struct {
	SeedKeywords []string `json:"seed_keywords,omitempty" jsonschema:"Seed keywords to generate ideas from (e.g. ['C# tutorial', 'dotnet performance']). At least one of seed_keywords or url must be provided."`
	URL          string   `json:"url,omitempty"           jsonschema:"A URL to generate ideas from (e.g. 'https://devleader.ca'). At least one of seed_keywords or url must be provided."`
	Language     string   `json:"language,omitempty"      jsonschema:"Language resource name (e.g. 'languageConstants/1000' for English). Omit to use all languages."`
}

// getHistoricalMetricsInput is the input schema for the get_historical_metrics tool.
type getHistoricalMetricsInput struct {
	Keywords []string `json:"keywords" jsonschema:"List of keywords to get historical search metrics for (e.g. ['dependency injection', 'SOLID principles'])."`
}

// getKeywordForecastInput is the input schema for the get_keyword_forecast tool.
type getKeywordForecastInput struct {
	Keywords     []string `json:"keywords"                 jsonschema:"List of keywords to forecast performance for."`
	MaxCPCMicros int64    `json:"max_cpc_micros,omitempty" jsonschema:"Maximum CPC bid in micros (1,000,000 = $1.00). Defaults to 1,000,000 if omitted or 0."`
	ForecastDays int      `json:"forecast_days,omitempty"  jsonschema:"Number of days to forecast. Defaults to 30 if omitted or 0."`
}

func generateKeywordIdeas(ctx context.Context, client *keywordplanner.Client, input generateKeywordIdeasInput) (*mcp.CallToolResult, any, error) {
	if len(input.SeedKeywords) == 0 && input.URL == "" {
		errResult := map[string]string{"error": "at least one of seed_keywords or url must be provided"}
		b, _ := json.Marshal(errResult)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	}
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
