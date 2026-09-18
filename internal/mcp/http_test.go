package mcp

import (
	"bytes"
	"context"
	"io"
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

const initializeRequestBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"db-mcp-test","version":"1.0.0"}}}`

// newTestHTTPTransport creates an HTTP transport with the endpoint and tokens needed by a test.
func newTestHTTPTransport(endpointPath string, tokens []string) *HTTPTransport {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{
		Transport:    "http",
		Host:         "localhost",
		Port:         8080,
		EndpointPath: endpointPath,
		Tokens:       tokens,
	}

	var auth *middleware.TokenAuth
	if len(tokens) > 0 {
		auth = middleware.NewTokenAuth(tokens)
	}
	return NewHTTPTransport(mcpServer, cfg, auth)
}

// performInitialize posts a real MCP initialize request to the supplied path.
func performInitialize(t *testing.T, client *http.Client, baseURL, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewBufferString(initializeRequestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

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
	transport := newTestHTTPTransport("/mcp", []string{"test-token"})

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

	// A valid token reaches the MCP endpoint, where bare GET is not supported.
	req, _ = http.NewRequest("GET", ts.URL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	resp.Body.Close()

	// Unknown paths must remain outside the protected MCP route.
	req, _ = http.NewRequest("GET", ts.URL+"/.well-known/oauth-protected-resource", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// A valid token must still allow MCP initialization.
	resp = performInitialize(t, client, ts.URL, "/mcp", "test-token")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestStreamableHTTPOAuthDiscoveryRequestsCompleteImmediately(t *testing.T) {
	transport := newTestHTTPTransport("/mcp", nil)
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()

	discoveryPaths := map[string]int{
		"/mcp": http.StatusMethodNotAllowed,
		"/.well-known/oauth-protected-resource/mcp":   http.StatusNotFound,
		"/mcp/.well-known/oauth-protected-resource":   http.StatusNotFound,
		"/.well-known/oauth-protected-resource":       http.StatusNotFound,
		"/.well-known/oauth-authorization-server/mcp": http.StatusNotFound,
		"/.well-known/openid-configuration/mcp":       http.StatusNotFound,
		"/mcp/.well-known/openid-configuration":       http.StatusNotFound,
		"/.well-known/oauth-authorization-server":     http.StatusNotFound,
	}

	for path, wantStatus := range discoveryPaths {
		t.Run(path, func(t *testing.T) {
			client := &http.Client{Timeout: 250 * time.Millisecond}
			started := time.Now()
			resp, err := client.Get(ts.URL + path)
			require.NoError(t, err)
			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err, "response body must terminate without waiting for client timeout")
			assert.Equal(t, wantStatus, resp.StatusCode)
			assert.NotEqual(t, "text/event-stream", resp.Header.Get("Content-Type"))
			assert.Less(t, time.Since(started), 200*time.Millisecond)
		})
	}
}

func TestStreamableHTTPRoutesOnlyConfiguredEndpoint(t *testing.T) {
	transport := newTestHTTPTransport("api/mcp/", nil)
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()
	client := ts.Client()

	resp := performInitialize(t, client, ts.URL, "/api/mcp", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	for _, path := range []string{"/mcp", "/api/mcp/", "/api/mcp/anything", "/api/mcp-extra", "/anything"} {
		t.Run(path, func(t *testing.T) {
			resp := performInitialize(t, client, ts.URL, path, "")
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestStreamableHTTPEmptyEndpointDefaultsToMCP(t *testing.T) {
	transport := newTestHTTPTransport("", nil)
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()
	client := ts.Client()

	resp := performInitialize(t, client, ts.URL, "/mcp", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = performInitialize(t, client, ts.URL, "/other", "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestStreamableHTTPRootEndpointDoesNotMatchSubpaths(t *testing.T) {
	transport := newTestHTTPTransport("/", nil)
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()
	client := ts.Client()

	resp := performInitialize(t, client, ts.URL, "/", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = performInitialize(t, client, ts.URL, "/anything", "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestStreamableHTTPUnknownPathsReturnNotFoundForAllMCPMethods(t *testing.T) {
	transport := newTestHTTPTransport("/mcp", nil)
	ts := httptest.NewServer(transport.httpServer.Handler)
	defer ts.Close()
	client := ts.Client()

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, ts.URL+"/unknown", bytes.NewBufferString(initializeRequestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestSSETransportKeepsItsOwnRoutes(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	cfg := &config.MCPConfig{Transport: "sse", Host: "localhost", Port: 8080}
	transport := NewHTTPTransport(mcpServer, cfg, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	transport.httpServer.Handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
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
