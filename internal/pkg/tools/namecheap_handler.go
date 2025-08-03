package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)



// NamecheapTool wraps a domain checking service for integration with the Model Context Protocol (MCP).
// It provides a standardized interface for domain availability checking tools within MCP applications.
type NamecheapTool struct {
	// service is the underlying domain checking service implementation
	service namecheap.DomainChecker
}

// NewNamecheapTool creates a new NamecheapTool wrapper around a domain checking service.
// The service parameter must implement the DomainChecker interface to provide
// domain availability checking functionality.
func NewNamecheapTool(service namecheap.DomainChecker) *NamecheapTool {
	return &NamecheapTool{
		service: service,
	}
}

// Name returns the name of the domain checking tool.
// This delegates to the underlying service's Name method to maintain consistency.
func (nt *NamecheapTool) Name() string {
	return nt.service.Name()
}

// Description returns a human-readable description of the domain checking tool.
// This delegates to the underlying service's Description method to maintain consistency.
func (nt *NamecheapTool) Description() string {
	return nt.service.Description()
}

// Handler processes domain availability checking requests via the Model Context Protocol.
// It accepts a list of domains to check and returns structured results with availability information,
// premium domain pricing, and associated fees. The response includes both structured data
// and JSON content.
func (nt *NamecheapTool) Handler(
	_ context.Context,
	_ *mcp.ServerSession,
	params *mcp.CallToolParamsFor[namecheap.ParamsIn],
) (*mcp.CallToolResultFor[namecheap.ParamsOut], error) {
	start := time.Now()

	defer func() {
		// Note: The service will handle logging internally
		_ = time.Since(start)
	}()

	results, err := nt.service.DomainsCheck(params.Arguments.Domains)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", namecheap.ErrNamecheapAPIFailed, err)
	}

	output := namecheap.ParamsOut{
		Results: results,
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("error marshaling results to JSON: %w", err)
	}

	return &mcp.CallToolResultFor[namecheap.ParamsOut]{
		StructuredContent: output,
		Meta:              map[string]interface{}{},
		IsError:           false,
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonData),
				Meta: map[string]interface{}{},
				Annotations: &mcp.Annotations{
					Audience:     []mcp.Role{"assistant"},
					LastModified: time.Now().Format(time.RFC3339),
					Priority:     1,
				},
			},
		},
	}, nil
}
