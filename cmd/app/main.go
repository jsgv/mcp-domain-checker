// Package main implements a simple MCP server that provides domain availability checking
// functionality using the Namecheap API.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	addr          = ":8080"
	serverName    = "com.jsgv.domain-checker"
	serverTitle   = "Domain Checker"
	version       = "0.0.1"
	serverTimeout = time.Minute * 3
)

func main() {
	var cfg config

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("Error parsing environment variables: ", err)
	}

	logger, err := createLogger(&cfg)
	if err != nil {
		log.Fatal("Error creating logger: ", err)
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   serverTitle,
		Version: version,
	}, nil)

	setupTools(mcpServer, logger, &cfg)

	startServer(mcpServer, logger)
}

func setupTools(mcpServer *mcp.Server, logger *zap.Logger, cfg *config) {
	// Add Namecheap tool if configuration is provided
	namecheapConfig := namecheap.Config{
		APIUser:  cfg.NamecheapAPIUser,
		APIKey:   cfg.NamecheapAPIKey,
		UserName: cfg.NamecheapUserName,
		ClientIP: cfg.NamecheapClientIP,
		Endpoint: cfg.NamecheapEndpoint,
	}

	if namecheapConfig.APIUser != "" && namecheapConfig.APIKey != "" &&
		namecheapConfig.UserName != "" && namecheapConfig.ClientIP != "" {
		namecheapTool, err := tools.GetNamecheapTool(logger, namecheapConfig)
		if err != nil {
			logger.Warn("Failed to create Namecheap tool", zap.Error(err))
		} else {
			mcp.AddTool(
				mcpServer,
				&mcp.Tool{ //nolint:exhaustruct
					Name:        namecheapTool.Name(),
					Description: namecheapTool.Description(),
				},
				namecheapTool.Handler,
			)
			logger.Info("Namecheap tool enabled")
		}
	} else {
		logger.Info("Namecheap tool disabled - missing configuration")
	}
}

func startServer(mcpServer *mcp.Server, logger *zap.Logger) {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	logger.Info("Starting server on " + addr)

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:        addr,
		Handler:     handler,
		ReadTimeout: serverTimeout,
	}

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
