package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// newTestServerAndClient wires a real in-memory MCP client/server pair around
// the actual production tool registrations (including any middleware added by
// registerMiddleware), backed by a fake Google Ads API. This exercises the full
// request pipeline -- schema validation, middleware, and the typed handler --
// exactly as a real MCP client would, rather than calling internal validation
// functions directly.
func newTestServerAndClient(t *testing.T, apiResponseBody string, registerMiddleware func(*mcp.Server)) *mcp.ClientSession {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiResponseBody))
	}))
	t.Cleanup(srv.Close)

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())

	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "test"}, nil)
	if registerMiddleware != nil {
		registerMiddleware(server)
	}
	mcp.AddTool(server,
		&mcp.Tool{Name: "generate_keyword_ideas", Description: "test"},
		func(ctx context.Context, _ *mcp.CallToolRequest, input generateKeywordIdeasInput) (*mcp.CallToolResult, any, error) {
			return generateKeywordIdeas(ctx, client, input)
		},
	)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

// TestGenerateKeywordIdeas_StringifiedSeedKeywords_Fails is the RED half of the
// TDD cycle for the "seed_keywords arrives as a JSON-encoded string" bug
// (google-keyword-planner-mcp#4). It reproduces the exact failure end-to-end,
// through the real schema-validation pipeline, with no middleware installed.
//
// Schema validation failures are surfaced as a normal CallToolResult with
// IsError=true (via CallToolResult.SetError), not as a protocol-level error --
// confirmed against go-sdk's toolForErr in server.go. This changed from a
// protocol-level error in go-sdk v1.5.0, specifically so validation problems
// are visible to the calling model in-band, the same channel as any other
// tool error, rather than as an opaque JSON-RPC error.
func TestGenerateKeywordIdeas_StringifiedSeedKeywords_Fails(t *testing.T) {
	t.Parallel()

	clientSession := newTestServerAndClient(t, `{"results": []}`, nil)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "generate_keyword_ideas",
		Arguments: map[string]any{
			// Simulates a client that double-encodes the array into a JSON string
			// before sending it, instead of a genuine JSON array.
			"seed_keywords": `["content marketing tools"]`,
			"url":           "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected protocol-level error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected a schema validation error result for stringified seed_keywords, got a successful result")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "seed_keywords") {
		t.Errorf("result text = %q, want it to mention seed_keywords", text)
	}
}

// TestGenerateKeywordIdeas_StringifiedSeedKeywords_CoercedByMiddleware is the
// GREEN half: with coerceStringifiedArrayArgs installed, the same stringified
// input is repaired before schema validation runs, and the call succeeds.
func TestGenerateKeywordIdeas_StringifiedSeedKeywords_CoercedByMiddleware(t *testing.T) {
	t.Parallel()

	clientSession := newTestServerAndClient(t, `{"results": []}`, func(s *mcp.Server) {
		s.AddReceivingMiddleware(coerceStringifiedArrayArgs(toolArrayFields))
	})

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "generate_keyword_ideas",
		Arguments: map[string]any{
			"seed_keywords": `["content marketing tools"]`,
			"url":           "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result content: %v", err)
	}
	if _, isError := parsed["error"]; isError {
		t.Errorf("result contains an error, want success: %v", parsed)
	}
}

// TestGenerateKeywordIdeas_GenuineArraySeedKeywords_StillWorks confirms the
// coercion middleware is a no-op for well-formed clients that already send a
// genuine JSON array -- it must not interfere with the standard-compliant path.
func TestGenerateKeywordIdeas_GenuineArraySeedKeywords_StillWorks(t *testing.T) {
	t.Parallel()

	clientSession := newTestServerAndClient(t, `{"results": []}`, func(s *mcp.Server) {
		s.AddReceivingMiddleware(coerceStringifiedArrayArgs(toolArrayFields))
	})

	_, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "generate_keyword_ideas",
		Arguments: map[string]any{
			"seed_keywords": []string{"content marketing tools"},
			"url":           "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error for a genuine JSON array: %v", err)
	}
}
