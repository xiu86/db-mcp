package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/connection"
	mcpserver "db-mcp/internal/mcp"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/server"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.NewLogger(&cfg.Log)

	log.Info("Starting db-mcp server", "version", version)

	// Connect to database
	connManager, err := connection.NewConnectionManager(cfg, log)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Health check
	if err := connManager.HealthCheck(); err != nil {
		log.Error("Database health check failed", "error", err)
		connManager.Close()
		os.Exit(1)
	}
	log.Info("Database connection established")

	// Create MCP server (this also initializes the audit service)
	mcpSvc := mcpserver.NewMCPServer(connManager, cfg, log)

	// Start server
	srv := mcpSvc.GetServer()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start stdio server in a goroutine so we can listen for signals
	go func() {
		log.Info("Starting MCP server on stdio")
		if err := server.ServeStdio(srv); err != nil {
			log.Error("Server error", "error", err)
		}
		// Signal main when stdio server exits (e.g., client disconnects)
		sigChan <- syscall.SIGTERM
	}()

	// Wait for signal
	<-sigChan

	log.Info("Shutting down server...")

	// Close MCP server (flushes audit log, etc.)
	if err := mcpSvc.Close(); err != nil {
		log.Error("Failed to close MCP server", "error", err)
	}

	// Close database connection (flushes buffers, closes connections)
	if err := connManager.Close(); err != nil {
		log.Error("Failed to close database connection", "error", err)
	} else {
		log.Info("Database connection closed")
	}

	// Give a moment for cleanup
	time.Sleep(100 * time.Millisecond)
	log.Info("Server stopped")
}

// HTTPHandler returns an HTTP handler for the MCP server (optional)
func HTTPHandler(cm *connection.ConnectionManager, cfg *config.Config, log *logger.Logger) http.Handler {
	_ = mcpserver.NewMCPServer(cm, cfg, log) // MCP server for HTTP transport
	return nil                                // SSE handler would be implemented here
}
