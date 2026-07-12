package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultHTTPListenAddress = "127.0.0.1"
	defaultHTTPPort          = 8080
	healthPath               = "/health"
	mcpPath                  = "/mcp"
	shutdownPath             = "/shutdown"
	maxMCPRequestBytes       = 1 << 20
)

type httpServerOptions struct {
	ListenAddress string
	Port          int
	AllowedHosts  []string
	ShutdownToken string
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Transport string `json:"transport"`
}

func runHTTP(ctx context.Context, srv *mcp.Server, options httpServerOptions) error {
	shutdownRequests := make(chan struct{}, 1)
	httpServer := newHTTPServer(srv, options, func() {
		select {
		case shutdownRequests <- struct{}{}:
		default:
		}
	})
	errorChannel := make(chan error, 1)

	slog.Info(
		"google-keyword-planner-mcp starting",
		"version",
		version,
		"transport",
		"http",
		"address",
		httpServer.Addr,
		"endpoint",
		mcpPath,
		"allowed_hosts",
		options.AllowedHosts,
	)

	go func() {
		errorChannel <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errorChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return shutdownHTTPServer(httpServer)
	case <-shutdownRequests:
		return shutdownHTTPServer(httpServer)
	}
}

func shutdownHTTPServer(server *http.Server) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return server.Shutdown(shutdownContext)
}

func newHTTPServer(
	srv *mcp.Server,
	options httpServerOptions,
	requestShutdown func(),
) *http.Server {
	return &http.Server{
		Addr: net.JoinHostPort(options.ListenAddress, strconv.Itoa(options.Port)),
		Handler: buildHTTPHandlerWithShutdown(
			srv,
			options.AllowedHosts,
			options.ShutdownToken,
			requestShutdown,
		),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
}

func buildHTTPHandler(srv *mcp.Server, allowedHosts []string) http.Handler {
	return buildHTTPHandlerWithShutdown(srv, allowedHosts, "", nil)
}

func buildHTTPHandlerWithShutdown(
	srv *mcp.Server,
	allowedHosts []string,
	shutdownToken string,
	requestShutdown func(),
) http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server {
			return srv
		},
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
	originProtection := http.NewCrossOriginProtection()

	mux := http.NewServeMux()
	mux.Handle(
		mcpPath,
		originProtection.Handler(http.MaxBytesHandler(mcpHandler, maxMCPRequestBytes)),
	)
	mux.HandleFunc("GET "+healthPath, serveHealth)
	if shutdownToken != "" && requestShutdown != nil {
		mux.HandleFunc("POST "+shutdownPath, func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			serveShutdown(writer, request, shutdownToken, requestShutdown)
		})
	}
	return allowedHostsMiddleware(mux, allowedHosts)
}

func serveHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(healthResponse{
		Status:    "ok",
		Service:   "google-keyword-planner-mcp",
		Version:   version,
		Transport: "http",
	}); err != nil {
		slog.Warn("failed to write health response", "err", err)
	}
}

func serveShutdown(
	writer http.ResponseWriter,
	request *http.Request,
	shutdownToken string,
	requestShutdown func(),
) {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	remoteIP := net.ParseIP(host)
	if err != nil || remoteIP == nil || !remoteIP.IsLoopback() {
		http.Error(writer, "shutdown is only available from loopback", http.StatusForbidden)
		return
	}

	const bearerPrefix = "Bearer "
	authorization := request.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, bearerPrefix) ||
		subtle.ConstantTimeCompare(
			[]byte(strings.TrimPrefix(authorization, bearerPrefix)),
			[]byte(shutdownToken),
		) != 1 {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusAccepted)
	_, _ = writer.Write([]byte(`{"stopping":true}`))
	requestShutdown()
}

func resolveHTTPListenAddress(flagValue string, explicitlySet bool) (string, error) {
	if explicitlySet {
		address := strings.TrimSpace(flagValue)
		if address == "" {
			return "", fmt.Errorf("listen address must not be empty")
		}
		return address, nil
	}
	if address := strings.TrimSpace(os.Getenv("MCP_LISTEN_ADDRESS")); address != "" {
		return address, nil
	}
	return defaultHTTPListenAddress, nil
}

func resolveHTTPPort(flagValue int, explicitlySet bool) (int, error) {
	if explicitlySet {
		return validateHTTPPort(flagValue)
	}
	if value := strings.TrimSpace(os.Getenv("PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("PORT must be an integer: %w", err)
		}
		return validateHTTPPort(port)
	}
	return defaultHTTPPort, nil
}

func validateHTTPPort(port int) (int, error) {
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return port, nil
}

func allowedHostsMiddleware(next http.Handler, allowedHosts []string) http.Handler {
	normalizedAllowedHosts := make([]string, 0, len(allowedHosts))
	for _, host := range allowedHosts {
		normalizedAllowedHosts = append(normalizedAllowedHosts, normalizeHost(host))
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		host := request.Host
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
		host = normalizeHost(host)

		allowed := false
		for _, allowedHost := range normalizedAllowedHosts {
			if strings.EqualFold(host, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(writer, "host not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func normalizeHost(host string) string {
	return strings.Trim(strings.TrimSpace(host), "[]")
}
