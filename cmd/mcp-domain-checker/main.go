// Package main implements a simple MCP server that provides domain availability checking
// functionality using the Namecheap API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"github.com/jsgv/mcp-domain-checker/internal/pkg/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	addr            = ":8080"
	serverName      = "com.jsgv.domain-checker"
	serverTitle     = "Domain Checker"
	serverTimeout   = time.Minute * 3
	shutdownTimeout = time.Second * 10

	transportHTTP  = "http"
	transportStdio = "stdio"
)

// errInvalidTransport is returned when the resolved transport value isn't one of
// the supported options. Declared as a sentinel so callers can errors.Is against it.
var errInvalidTransport = errors.New("invalid transport")

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

	transportFlag := flag.String("transport", "",
		"Transport: http or stdio (overrides TRANSPORT env; default http)")

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

	transport, err := resolveTransport(*transportFlag, cfg.Transport)
	if err != nil {
		log.Fatal("Error resolving transport: ", err)
	}

	logger, err := createLogger(&cfg)
	if err != nil {
		log.Fatal("Error creating logger: ", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mcpServer := mcp.NewServer(&mcp.Implementation{ //nolint:exhaustruct
		Name:    serverName,
		Title:   serverTitle,
		Version: version,
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{}, //nolint:exhaustruct
	})

	setupTools(mcpServer, logger, &cfg)

	switch transport {
	case transportStdio:
		runStdio(ctx, mcpServer, logger)
	case transportHTTP:
		runHTTP(ctx, mcpServer, logger)
	}
}

// resolveTransport picks the transport to use. A non-empty flag value wins
// over the env-derived value. Only "http" and "stdio" are accepted.
func resolveTransport(flagVal, envVal string) (string, error) {
	value := flagVal
	if value == "" {
		value = envVal
	}

	switch value {
	case transportHTTP, transportStdio:
		return value, nil
	default:
		return "", fmt.Errorf("%w %q: must be %q or %q",
			errInvalidTransport, value, transportHTTP, transportStdio)
	}
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

func runStdio(ctx context.Context, mcpServer *mcp.Server, logger *zap.Logger) {
	logger.Info("Starting stdio transport")

	err := mcpServer.Run(ctx, &mcp.StdioTransport{})
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("stdio transport exited with error", zap.Error(err))
	}
}

func runHTTP(ctx context.Context, mcpServer *mcp.Server, logger *zap.Logger) {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	corsHandler := corsMiddleware(handler)

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:        addr,
		Handler:     corsHandler,
		ReadTimeout: serverTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("Starting server on " + addr)

		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("Shutting down HTTP server")

		// Fresh context: the parent is already cancelled, so using it would
		// abort Shutdown immediately instead of letting in-flight requests drain.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := httpServer.Shutdown(shutdownCtx) //nolint:contextcheck
		if err != nil {
			logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("ListenAndServe", zap.Error(err))
		}
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
