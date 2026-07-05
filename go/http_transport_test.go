package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// TestAllowedHostsMiddleware_AllowedHost_PassesThrough verifies a request whose
// Host header matches the allow-list reaches the wrapped handler.
func TestAllowedHostsMiddleware_AllowedHost_PassesThrough(t *testing.T) {
	t.Parallel()

	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	handler := allowedHostsMiddleware(next, []string{"localhost", "127.0.0.1"})

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !reached {
		t.Error("request with allowed host must reach the wrapped handler")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestAllowedHostsMiddleware_DisallowedHost_Rejected verifies a request whose
// Host header is absent from the allow-list is rejected before the wrapped
// handler runs, defending against DNS rebinding.
func TestAllowedHostsMiddleware_DisallowedHost_Rejected(t *testing.T) {
	t.Parallel()

	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	handler := allowedHostsMiddleware(next, []string{"localhost", "127.0.0.1"})

	req := httptest.NewRequest(http.MethodPost, "http://evil.example.com/", nil)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if reached {
		t.Error("request with disallowed host must not reach the wrapped handler")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestHTTPTransport_ServesRealSession exercises the full HTTP transport stack --
// allowedHostsMiddleware wrapping mcp.NewStreamableHTTPHandler wrapping the real
// newServer(client) -- through a real MCP client connecting over HTTP, the same
// way runHTTP wires them in production (minus the actual port bind, since the
// test uses httptest.Server instead of http.ListenAndServe).
func TestHTTPTransport_ServesRealSession(t *testing.T) {
	t.Parallel()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer apiSrv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", apiSrv.URL, apiSrv.Client())
	srv := newServer(client)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	httpSrv := httptest.NewServer(allowedHostsMiddleware(mcpHandler, []string{"127.0.0.1"}))
	defer httpSrv.Close()

	ctx := context.Background()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := mcpClient.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpSrv.URL}, nil)
	if err != nil {
		t.Fatalf("client.Connect over HTTP: %v", err)
	}
	defer session.Close()

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools over HTTP: %v", err)
	}
	if len(toolsResult.Tools) != 3 {
		t.Errorf("got %d tools over HTTP, want 3", len(toolsResult.Tools))
	}

	callResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_keyword_ideas",
		Arguments: map[string]any{"seed_keywords": []string{"go programming"}},
	})
	if err != nil {
		t.Fatalf("CallTool over HTTP: %v", err)
	}
	if callResult.IsError {
		t.Errorf("CallTool over HTTP returned an error result: %+v", callResult.Content)
	}
}

// TestHTTPTransport_RejectsDisallowedHost verifies the allow-list is actually
// wired into the served stack, not just unit-testable in isolation: a request
// carrying a disallowed Host header never reaches the MCP handler at all.
func TestHTTPTransport_RejectsDisallowedHost(t *testing.T) {
	t.Parallel()

	client := keywordplanner.NewTestClient("dev-token", "123", "", "http://unused.invalid", http.DefaultClient)
	srv := newServer(client)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	httpSrv := httptest.NewServer(allowedHostsMiddleware(mcpHandler, []string{"only-this-host-is-allowed"}))
	defer httpSrv.Close()

	resp, err := http.Post(httpSrv.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// TestSplitAndTrim_ParsesCommaSeparatedList confirms the --allowed-hosts flag
// value is parsed into a clean slice: trimmed, with empty entries dropped.
func TestSplitAndTrim_ParsesCommaSeparatedList(t *testing.T) {
	t.Parallel()

	got := splitAndTrim("localhost, 127.0.0.1 ,, [::1]")
	want := []string{"localhost", "127.0.0.1", "[::1]"}

	if len(got) != len(want) {
		t.Fatalf("splitAndTrim(...) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitAndTrim(...)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
