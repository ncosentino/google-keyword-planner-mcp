package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func TestAllowedHostsMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		host       string
		wantStatus int
		wantCalled bool
	}{
		{name: "allowed", host: "127.0.0.1:8080", wantStatus: http.StatusOK, wantCalled: true},
		{name: "rejected", host: "evil.example", wantStatus: http.StatusForbidden, wantCalled: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := false
			handler := allowedHostsMiddleware(
				http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					called = true
					writer.WriteHeader(http.StatusOK)
				}),
				[]string{"127.0.0.1"},
			)
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/", nil)
			request.Host = test.host
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if called != test.wantCalled {
				t.Errorf("handler called = %v, want %v", called, test.wantCalled)
			}
		})
	}
}

func TestHTTPTransport_ServesRealSession(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"results":[]}`))
	}))
	defer apiServer.Close()

	client := keywordplanner.NewTestClient(
		"dev-token",
		"123",
		"",
		apiServer.URL,
		apiServer.Client(),
	)
	server := newServer(client)
	httpServer := httptest.NewServer(buildHTTPHandler(server, []string{"127.0.0.1"}))
	defer httpServer.Close()

	ctx := context.Background()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := mcpClient.Connect(
		ctx,
		&mcp.StreamableClientTransport{Endpoint: httpServer.URL + mcpPath},
		nil,
	)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 3 {
		t.Errorf("tools = %d, want 3", len(tools.Tools))
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "generate_keyword_ideas",
		Arguments: map[string]any{
			"seed_keywords": []string{"go programming"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Errorf("CallTool returned error content: %+v", result.Content)
	}
}

func TestHTTPTransport_ServesHealth(t *testing.T) {
	t.Parallel()

	server := newServer(keywordplanner.NewTestClient(
		"dev-token",
		"123",
		"",
		"http://unused.invalid",
		http.DefaultClient,
	))
	httpServer := httptest.NewServer(buildHTTPHandler(server, []string{"127.0.0.1"}))
	defer httpServer.Close()

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		httpServer.URL+healthPath,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var health healthResponse
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health.Status != "ok" || health.Service != "google-keyword-planner-mcp" {
		t.Errorf("health = %+v", health)
	}
}

func TestHTTPTransport_RejectsForgedCrossSiteOrigin(t *testing.T) {
	t.Parallel()

	server := newServer(keywordplanner.NewTestClient(
		"dev-token",
		"123",
		"",
		"http://unused.invalid",
		http.DefaultClient,
	))
	httpServer := httptest.NewServer(buildHTTPHandler(server, []string{"127.0.0.1"}))
	defer httpServer.Close()

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		httpServer.URL+mcpPath,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", response.StatusCode)
	}
}

func TestHTTPTransport_ShutdownRequiresLoopbackBearerToken(t *testing.T) {
	t.Parallel()

	shutdownRequested := false
	handler := buildHTTPHandlerWithShutdown(
		newServer(keywordplanner.NewTestClient(
			"dev-token",
			"123",
			"",
			"http://unused.invalid",
			http.DefaultClient,
		)),
		[]string{"127.0.0.1"},
		"secret-token",
		func() { shutdownRequested = true },
	)

	tests := []struct {
		name       string
		remoteAddr string
		token      string
		wantStatus int
		wantStop   bool
	}{
		{
			name:       "remote caller",
			remoteAddr: "192.0.2.10:1234",
			token:      "secret-token",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "wrong token",
			remoteAddr: "127.0.0.1:1234",
			token:      "wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "authorized loopback caller",
			remoteAddr: "127.0.0.1:1234",
			token:      "secret-token",
			wantStatus: http.StatusAccepted,
			wantStop:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shutdownRequested = false
			request := httptest.NewRequest(
				http.MethodPost,
				"http://127.0.0.1"+shutdownPath,
				nil,
			)
			request.Host = "127.0.0.1"
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("Authorization", "Bearer "+test.token)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if shutdownRequested != test.wantStop {
				t.Errorf("shutdown requested = %v, want %v", shutdownRequested, test.wantStop)
			}
		})
	}
}

func TestNewHTTPServer_DefaultsToLoopbackAddress(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(
		newServer(keywordplanner.NewTestClient(
			"dev-token",
			"123",
			"",
			"http://unused.invalid",
			http.DefaultClient,
		)),
		httpServerOptions{
			ListenAddress: defaultHTTPListenAddress,
			Port:          defaultHTTPPort,
			AllowedHosts:  []string{"127.0.0.1"},
		},
		nil,
	)
	if server.Addr != "127.0.0.1:8080" {
		t.Errorf("address = %q, want 127.0.0.1:8080", server.Addr)
	}
	if server.ReadHeaderTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Errorf(
			"timeouts = read header %s, idle %s; both must be positive",
			server.ReadHeaderTimeout,
			server.IdleTimeout,
		)
	}
}

func TestResolveHTTPPort(t *testing.T) {
	t.Setenv("PORT", "9999")
	got, err := resolveHTTPPort(0, false)
	if err != nil {
		t.Fatalf("resolveHTTPPort: %v", err)
	}
	if got != 9999 {
		t.Errorf("port = %d, want 9999", got)
	}

	t.Setenv("PORT", "")
	got, err = resolveHTTPPort(0, false)
	if err != nil {
		t.Fatalf("resolveHTTPPort: %v", err)
	}
	if got != defaultHTTPPort {
		t.Errorf("port = %d, want %d", got, defaultHTTPPort)
	}

	got, err = resolveHTTPPort(9000, true)
	if err != nil {
		t.Fatalf("resolveHTTPPort flag: %v", err)
	}
	if got != 9000 {
		t.Errorf("flag port = %d, want 9000", got)
	}

	if _, err := resolveHTTPPort(0, true); err == nil {
		t.Fatal("explicit zero port returned nil error")
	}
}

func TestResolveHTTPPort_RejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("PORT", "invalid")
	if _, err := resolveHTTPPort(0, false); err == nil {
		t.Fatal("resolveHTTPPort returned nil error")
	}
}

func TestResolveHTTPListenAddress(t *testing.T) {
	t.Setenv("MCP_LISTEN_ADDRESS", "192.0.2.10")
	got, err := resolveHTTPListenAddress("", false)
	if err != nil {
		t.Fatalf("resolveHTTPListenAddress: %v", err)
	}
	if got != "192.0.2.10" {
		t.Errorf("address = %q, want 192.0.2.10", got)
	}
	got, err = resolveHTTPListenAddress("127.0.0.2", true)
	if err != nil {
		t.Fatalf("resolveHTTPListenAddress flag: %v", err)
	}
	if got != "127.0.0.2" {
		t.Errorf("flag address = %q, want 127.0.0.2", got)
	}
	if _, err := resolveHTTPListenAddress("", true); err == nil {
		t.Fatal("explicit empty address returned nil error")
	}
}

func TestAllowedHostsMiddleware_NormalizesIPv6(t *testing.T) {
	t.Parallel()

	called := false
	handler := allowedHostsMiddleware(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			called = true
			writer.WriteHeader(http.StatusOK)
		}),
		[]string{"[::1]"},
	)
	request := httptest.NewRequest(http.MethodGet, "http://[::1]/health", nil)
	request.Host = "[::1]:8080"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !called {
		t.Errorf("status = %d, called = %v; want 200 and true", recorder.Code, called)
	}
}

func TestSplitAndTrim(t *testing.T) {
	t.Parallel()

	got := splitAndTrim("localhost, 127.0.0.1 ,, [::1]")
	want := []string{"localhost", "127.0.0.1", "[::1]"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("got[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
