// Package tools provides a unified interface for domain checking tools.
package tools

import (
	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"go.uber.org/zap"
)

// DomainToolsFactory provides a factory for creating various domain checking tools.
// It maintains a logger instance that is passed to all created tools for consistent logging.
type DomainToolsFactory struct {
	// logger is used by all domain checking tools created by this factory
	logger *zap.Logger
}

// NewDomainChecker creates a new domain tools factory with the specified logger.
// The logger will be used by all domain checking tools created through this factory.
func NewDomainChecker(logger *zap.Logger) *DomainToolsFactory {
	return &DomainToolsFactory{
		logger: logger,
	}
}

// GetNamecheapTool creates and returns a Namecheap domain checking tool with the given configuration.
// This is a standalone convenience function that creates the tool directly without requiring a DomainChecker instance.
// It validates the configuration and returns an error if required credentials are missing.
func GetNamecheapTool(logger *zap.Logger, config namecheap.Config) (*NamecheapTool, error) {
	service, err := namecheap.NewNamecheapTool(logger, config)
	if err != nil {
		return nil, err
	}

	return NewNamecheapTool(service), nil
}

// NewNamecheapTool creates a new Namecheap domain checking tool using the factory's logger.
// It validates the provided configuration and returns an error if required credentials are missing.
func (dtf *DomainToolsFactory) NewNamecheapTool(config namecheap.Config) (*NamecheapTool, error) {
	service, err := namecheap.NewNamecheapTool(dtf.logger, config)
	if err != nil {
		return nil, err
	}

	return NewNamecheapTool(service), nil
}

