package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/middleware"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/server"
)

// HTTPTransport encapsulates an HTTP-based MCP transport
type HTTPTransport struct {
	httpServer *http.Server
	logger     *logger.Logger
}

// NewHTTPTransport creates a new HTTP transport for the MCP server
func NewHTTPTransport(mcpServer *server.MCPServer, cfg *config.MCPConfig, auth *middleware.TokenAuth) *HTTPTransport {
	var handler http.Handler

	switch cfg.Transport {
	case "sse":
		sseServer := server.NewSSEServer(mcpServer)
		handler = http.Handler(sseServer)
		if auth != nil && len(cfg.Tokens) > 0 {
			handler = auth.Middleware(handler)
		}
	default:
		handler = newStreamableHTTPHandler(mcpServer, cfg, auth)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // streamable HTTP needs long write timeout
		IdleTimeout:  120 * time.Second,
	}

	return &HTTPTransport{
		httpServer: httpSrv,
		logger:     nil,
	}
}

// newStreamableHTTPHandler mounts MCP on exactly one endpoint and keeps discovery paths outside auth.
func newStreamableHTTPHandler(mcpServer *server.MCPServer, cfg *config.MCPConfig, auth *middleware.TokenAuth) http.Handler {
	endpointPath := normalizeEndpointPath(cfg.EndpointPath)
	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		// db-mcp does not initiate server-to-client requests, so the optional GET stream is unnecessary.
		server.WithDisableStreaming(true),
	)

	var mcpHandler http.Handler = streamableServer
	if auth != nil && len(cfg.Tokens) > 0 {
		mcpHandler = auth.Middleware(mcpHandler)
	}

	mux := http.NewServeMux()
	mux.Handle(endpointPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointPath {
			http.NotFound(w, r)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	}))
	return mux
}

// normalizeEndpointPath applies the documented /mcp default and removes surrounding slashes.
func normalizeEndpointPath(endpointPath string) string {
	trimmedPath := strings.Trim(endpointPath, "/")
	if trimmedPath == "" {
		if endpointPath == "/" {
			return "/"
		}
		return "/mcp"
	}
	return "/" + trimmedPath
}

// Start begins listening for HTTP requests
func (t *HTTPTransport) Start(log *logger.Logger) error {
	t.logger = log
	t.logger.Info("Starting HTTP transport", "addr", t.httpServer.Addr, "transport", "http")
	return t.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server
func (t *HTTPTransport) Shutdown(ctx context.Context) error {
	if t.logger != nil {
		t.logger.Info("Shutting down HTTP transport")
	}
	return t.httpServer.Shutdown(ctx)
}
