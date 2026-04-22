package main

import (
	"context"
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
	"db-mcp/internal/middleware"
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
	transport := cfg.MCP.Transport
	if transport == "" {
		transport = "stdio" // default
	}
	// In stdio mode, stdout must be reserved for MCP JSON-RPC frames.
	if transport == "stdio" {
		cfg.Log.Output = "stderr"
	}
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
	mcpSvc, err := mcpserver.NewMCPServer(connManager, cfg, log)
	if err != nil {
		log.Error("Failed to create MCP server", "error", err)
		connManager.Close()
		os.Exit(1)
	}

	srv := mcpSvc.GetServer()

	switch transport {
	case "http", "sse", "streamable-http":
		// Create HTTP transport
		auth := middleware.NewTokenAuth(cfg.MCP.Tokens)
		httpTransport := mcpserver.NewHTTPTransport(srv, &cfg.MCP, auth)

		// Graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		go func() {
			if err := httpTransport.Start(log); err != nil && err != http.ErrServerClosed {
				log.Error("HTTP server error", "error", err)
			}
		}()

		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		httpTransport.Shutdown(shutdownCtx)

	default: // "stdio"
		log.Info("Starting MCP server in stdio mode")
		if err := server.ServeStdio(srv); err != nil {
			log.Error("Stdio server error", "error", err)
		}
	}

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
