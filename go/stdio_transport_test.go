package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// TestStdioTransport_ServesRealSession is a characterization test for the
// default (stdio) transport path, written ahead of the go-sdk dependency
// upgrade (issue #10). Production main() drives this path via
// srv.Run(ctx, &mcp.StdioTransport{}), but mcp.StdioTransport hardcodes
// os.Stdin/os.Stdout, so it cannot be pointed at a test double directly.
// StdioTransport.Connect is, however, byte-for-byte mcp.IOTransport.Connect
// with os.Stdin/os.Stdout substituted in -- same newline-delimited JSON
// framing, same connection type -- so wiring newServer(client) through
// IOTransport over real in-process pipes exercises the identical
// framing/protocol code stdio uses in production, without spawning a
// subprocess. Before this test, nothing automated exercised the stdio code
// path at all: every other test used either mcp.NewInMemoryTransports or the
// HTTP transport.
func TestStdioTransport_ServesRealSession(t *testing.T) {
	t.Parallel()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer apiSrv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", apiSrv.URL, apiSrv.Client())
	srv := newServer(client)

	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()

	ctx := context.Background()

	serverSession, err := srv.Connect(ctx, &mcp.IOTransport{Reader: serverRead, Writer: serverWrite}, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := mcpClient.Connect(ctx, &mcp.IOTransport{Reader: clientRead, Writer: clientWrite}, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	toolsResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools over stdio-equivalent transport: %v", err)
	}
	if len(toolsResult.Tools) != 3 {
		t.Errorf("got %d tools over stdio-equivalent transport, want 3", len(toolsResult.Tools))
	}

	callResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_keyword_ideas",
		Arguments: map[string]any{"seed_keywords": []string{"go programming"}},
	})
	if err != nil {
		t.Fatalf("CallTool over stdio-equivalent transport: %v", err)
	}
	if callResult.IsError {
		t.Errorf("CallTool over stdio-equivalent transport returned an error result: %+v", callResult.Content)
	}
}
