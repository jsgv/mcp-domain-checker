// Package main implements a simple MCP server that provides domain availability checking
// functionality using the Namecheap API.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	addr          = ":8080"
	serverName    = "com.jsgv.domain-checker"
	serverTitle   = "Domain Checker"
	serverTimeout = time.Minute * 3
)

// version and commit are set at build time via -ldflags.
//
//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.BoolVar(showVersion, "v", false, "Print version and exit (shorthand)")

	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintf(os.Stdout, "mcp-domain-checker version %s (commit: %s)\n", version, commit)

		os.Exit(0)
	}

	var cfg config

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("Error parsing environment variables: ", err)
	}

	logger, err := createLogger(&cfg)
	if err != nil {
		log.Fatal("Error creating logger: ", err)
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{ //nolint:exhaustruct
		Name:    serverName,
		Title:   serverTitle,
		Version: version,
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{}, //nolint:exhaustruct
	})

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
		service, err := namecheap.NewService(logger, namecheapConfig)
		if err != nil {
			logger.Warn("Failed to create Namecheap service", zap.Error(err))
		} else {
			namecheapTool := tool.NewTool(service)
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

	corsHandler := corsMiddleware(handler)

	logger.Info("Starting server on " + addr)

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:        addr,
		Handler:     corsHandler,
		ReadTimeout: serverTimeout,
	}

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Session-Id")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)

			return
		}

		next.ServeHTTP(w, r)
	})
}
