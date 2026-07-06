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

// TestHTTPTransport_ServesRealSession exercises the full HTTP transport stack
// via buildHTTPHandler (the same composition runHTTP serves in production:
// cross-origin protection wrapping allowedHostsMiddleware wrapping
// mcp.NewStreamableHTTPHandler wrapping the real newServer(client)) through a
// real MCP client connecting over HTTP (minus the actual port bind, since the
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

	httpSrv := httptest.NewServer(buildHTTPHandler(srv, []string{"127.0.0.1"}))
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

	httpSrv := httptest.NewServer(buildHTTPHandler(srv, []string{"only-this-host-is-allowed"}))
	defer httpSrv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, httpSrv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// TestHTTPTransport_RejectsForgedCrossSiteOrigin verifies cross-origin (CSRF)
// protection is actually wired into the served stack: a request with a
// forged, mismatched Origin header and a browser-style Sec-Fetch-Site header
// is rejected even though its Host header is on the allow-list -- simulating
// a malicious web page's fetch() call against a locally-running instance of
// this server, which allowedHostsMiddleware alone (a Host-header check) does
// not defend against.
func TestHTTPTransport_RejectsForgedCrossSiteOrigin(t *testing.T) {
	t.Parallel()

	client := keywordplanner.NewTestClient("dev-token", "123", "", "http://unused.invalid", http.DefaultClient)
	srv := newServer(client)

	httpSrv := httptest.NewServer(buildHTTPHandler(srv, []string{"127.0.0.1"}))
	defer httpSrv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, httpSrv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// TestHTTPTransport_AllowsSameOriginRequest verifies cross-origin protection
// does not block legitimate same-origin browser requests: a request whose
// Sec-Fetch-Site header says "same-origin" reaches the MCP handler (and gets
// past its own Content-Type/session validation instead of being stopped
// earlier by a 403).
func TestHTTPTransport_AllowsSameOriginRequest(t *testing.T) {
	t.Parallel()

	client := keywordplanner.NewTestClient("dev-token", "123", "", "http://unused.invalid", http.DefaultClient)
	srv := newServer(client)

	httpSrv := httptest.NewServer(buildHTTPHandler(srv, []string{"127.0.0.1"}))
	defer httpSrv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, httpSrv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		t.Errorf("status = %d, same-origin request must not be rejected by cross-origin protection", resp.StatusCode)
	}
}

// TestResolveHTTPPort_UsesEnvVarWhenSet confirms PORT, when set, takes
// precedence -- required for platforms (e.g. Cloud Run) that assign it
// automatically.
func TestResolveHTTPPort_UsesEnvVarWhenSet(t *testing.T) {
	t.Setenv("PORT", "9999")

	if got := resolveHTTPPort(); got != "9999" {
		t.Errorf("resolveHTTPPort() = %q, want %q", got, "9999")
	}
}

// TestResolveHTTPPort_DefaultsTo8080WhenUnset confirms the documented default
// of 8080 applies when PORT is unset, forced deterministically via t.Setenv
// rather than relying on the ambient environment not already having PORT set.
func TestResolveHTTPPort_DefaultsTo8080WhenUnset(t *testing.T) {
	t.Setenv("PORT", "")

	if got := resolveHTTPPort(); got != "8080" {
		t.Errorf("resolveHTTPPort() = %q, want %q", got, "8080")
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
