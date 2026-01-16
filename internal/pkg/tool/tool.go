// Package tool provides generic MCP tool wrappers.
package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service defines a generic interface for MCP tool services.
type Service[In, Out any] interface {
	// Name returns the unique identifier name of the service.
	Name() string
	// Description returns a human-readable description of the service.
	Description() string
	// Execute performs the service operation with the given input.
	Execute(in In) (Out, error)
}

// Tool wraps a service for integration with the Model Context Protocol (MCP).
type Tool[In, Out any] struct {
	service Service[In, Out]
}

// NewTool creates a new Tool wrapper around a service.
func NewTool[In, Out any](service Service[In, Out]) *Tool[In, Out] {
	return &Tool[In, Out]{
		service: service,
	}
}

// Name returns the name of the tool.
func (t *Tool[In, Out]) Name() string {
	return t.service.Name()
}

// Description returns a human-readable description of the tool.
func (t *Tool[In, Out]) Description() string {
	return t.service.Description()
}

// Handler processes requests via the Model Context Protocol.
func (t *Tool[In, Out]) Handler( //nolint:ireturn
	_ context.Context,
	_ *mcp.CallToolRequest,
	args In,
) (*mcp.CallToolResult, Out, error) {
	var zero Out

	output, err := t.service.Execute(args)
	if err != nil {
		return nil, zero, err
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return nil, zero, fmt.Errorf("error marshaling results to JSON: %w", err)
	}

	return &mcp.CallToolResult{ //nolint:exhaustruct
		IsError: false,
		Content: []mcp.Content{
			&mcp.TextContent{ //nolint:exhaustruct
				Text: string(jsonData),
				Annotations: &mcp.Annotations{ //nolint:exhaustruct
					Audience: []mcp.Role{"assistant"},
					Priority: 1,
				},
			},
		},
	}, output, nil
}
