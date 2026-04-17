package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/middleware"

	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPTransport(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         8080,
		EndpointPath: "/mcp",
		Tokens:       []string{},
	}

	transport := NewHTTPTransport(mcpServer, cfg, nil)

	require.NotNil(t, transport)
	require.NotNil(t, transport.httpServer)
	assert.Equal(t, "localhost:8080", transport.httpServer.Addr)
}

func TestNewHTTPTransportWithSSE(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "sse",
		Host:         "0.0.0.0",
		Port:         9090,
		EndpointPath: "/sse",
		Tokens:       []string{},
	}

	transport := NewHTTPTransport(mcpServer, cfg, nil)

	require.NotNil(t, transport)
	require.NotNil(t, transport.httpServer)
	assert.Equal(t, "0.0.0.0:9090", transport.httpServer.Addr)
}

func TestNewHTTPTransportWithAuth(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         8080,
		EndpointPath: "/mcp",
		Tokens:       []string{"test-token"},
	}

	auth := middleware.NewTokenAuth(cfg.Tokens)
	transport := NewHTTPTransport(mcpServer, cfg, auth)

	require.NotNil(t, transport)
	require.NotNil(t, transport.httpServer)

	// Test that auth middleware is working
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()

	// Request without token should be unauthorized
	req, _ := http.NewRequest("GET", ts.URL+"/mcp", nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Request with valid token should pass
	req, _ = http.NewRequest("GET", ts.URL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = client.Do(req)
	require.NoError(t, err)
	// The endpoint might not exist, but auth should pass
	// We expect either 404 (endpoint not found) or 200/405 (method not allowed)
	// but NOT 401 (unauthorized)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestNewHTTPTransportWithNilAuth(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         8080,
		EndpointPath: "/mcp",
		Tokens:       []string{},
	}

	transport := NewHTTPTransport(mcpServer, cfg, nil)

	require.NotNil(t, transport)
	require.NotNil(t, transport.httpServer)
}

func TestHTTPTransportShutdown(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         0, // Use random port for testing
		EndpointPath: "/mcp",
		Tokens:       []string{},
	}

	transport := NewHTTPTransport(mcpServer, cfg, nil)

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- transport.httpServer.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify server stopped
	select {
	case err := <-serverErr:
		assert.Error(t, err) // Expected error after shutdown
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not shut down")
	}
}

func TestHTTPTransportTimeouts(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         8080,
		EndpointPath: "/mcp",
		Tokens:       []string{},
	}

	transport := NewHTTPTransport(mcpServer, cfg, nil)

	assert.Equal(t, 30*time.Second, transport.httpServer.ReadTimeout)
	assert.Equal(t, time.Duration(0), transport.httpServer.WriteTimeout)
	assert.Equal(t, 120*time.Second, transport.httpServer.IdleTimeout)
}
