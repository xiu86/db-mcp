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
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"
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
	defer connManager.Close()

	// Health check
	if err := connManager.HealthCheck(); err != nil {
		log.Error("Database health check failed", "error", err)
		os.Exit(1)
	}
	log.Info("Database connection established")

	// Create MCP server
	mcpSvc := mcpserver.NewMCPServer(connManager.DB(), cfg, log)

	// Start server
	srv := mcpSvc.GetServer()

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = ctx
		log.Info("Server stopped")
		os.Exit(0)
	}()

	// Start stdio server
	log.Info("Starting MCP server on stdio")
	if err := server.ServeStdio(srv); err != nil {
		log.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// HTTPHandler returns an HTTP handler for the MCP server (optional)
func HTTPHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger) http.Handler {
	_ = mcpserver.NewMCPServer(db, cfg, log) // MCP server for HTTP transport
	return nil                                // SSE handler would be implemented here
}
