package main

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runHTTP serves srv over the MCP Streamable HTTP transport, listening on the
// PORT environment variable (default 8080). Requests whose Host header isn't in
// allowedHosts are rejected before reaching the MCP handler.
func runHTTP(srv *mcp.Server, allowedHosts []string) {
	port := resolveHTTPPort()

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, &mcp.StreamableHTTPOptions{
		// This server has no need for server-to-client requests, so stateless mode
		// is the documented recommendation: no session-affinity requirements, and
		// no in-memory session state to leak across requests or restarts.
		Stateless: true,
	})

	protected := allowedHostsMiddleware(handler, allowedHosts)

	slog.Info("google-keyword-planner-mcp starting",
		"version", version, "transport", "http", "port", port, "allowed_hosts", allowedHosts)
	if err := http.ListenAndServe(":"+port, protected); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

// resolveHTTPPort returns the PORT environment variable's value, or "8080" if
// it is unset or empty. Some deployment platforms (e.g. Cloud Run) set PORT
// automatically, so this must be read at call time rather than assumed absent.
func resolveHTTPPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

// allowedHostsMiddleware rejects requests whose Host header (ignoring any port)
// doesn't match an entry in allowedHosts, with 403 Forbidden. net/http, unlike
// some other server frameworks, does not validate the Host header on its own,
// which otherwise leaves an HTTP-transport deployment open to DNS rebinding:
// a browser could reach a server bound to localhost via an attacker-controlled
// DNS name that resolves to 127.0.0.1, bypassing same-origin protections.
func allowedHostsMiddleware(next http.Handler, allowedHosts []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if !slices.Contains(allowedHosts, host) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
