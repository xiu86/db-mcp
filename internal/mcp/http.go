package mcp

import (
	"context"
	"fmt"
	"net/http"
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
		// SSE transport with optional auth
		sseServer := server.NewSSEServer(mcpServer)
		handler = http.Handler(sseServer)
	default:
		// Streamable HTTP (default for "http" or "streamable-http")
		opts := []server.StreamableHTTPOption{}
		if cfg.EndpointPath != "" {
			opts = append(opts, server.WithEndpointPath(cfg.EndpointPath))
		}
		httpServer := server.NewStreamableHTTPServer(mcpServer, opts...)
		handler = httpServer
	}

	// Wrap with auth middleware if tokens are configured
	if auth != nil && len(cfg.Tokens) > 0 {
		handler = auth.Middleware(handler)
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
