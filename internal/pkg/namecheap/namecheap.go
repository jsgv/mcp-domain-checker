// Package namecheap provides domain availability checking using the Namecheap API.
package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	// maxDomainsPerCheck is the maximum number of domains allowed in a single API request.
	maxDomainsPerCheck = 50
	// httpTimeoutSeconds is the timeout for HTTP requests in seconds.
	httpTimeoutSeconds = 30
)

var (
	// ErrMissingDomains is returned when no domains are provided for checking.
	ErrMissingDomains        = errors.New("missing domains to check")
	// ErrMissingAPICredentials is returned when required API credentials are missing.
	ErrMissingAPICredentials = errors.New("missing API credentials")
	// ErrNamecheapAPIFailed is returned when the Namecheap API call fails.
	ErrNamecheapAPIFailed    = errors.New("Namecheap API call failed")
	// ErrAPIError is returned when the API returns an error response.
	ErrAPIError              = errors.New("API error")
	// ErrMaxDomainsExceeded is returned when more than 50 domains are requested.
	ErrMaxDomainsExceeded    = errors.New("max of 50 domains are allowed in a single check command")
)

// DomainChecker defines the interface for domain availability checking services.
// Implementations must provide methods to check domains and return service metadata.
type DomainChecker interface {
	// DomainsCheck checks domain availability for the given list of domains.
	// Returns a slice of Result with availability information for each domain.
	DomainsCheck(domains []string) ([]Result, error)
	// Name returns the unique identifier name of the service.
	Name() string
	// Description returns a human-readable description of the service.
	Description() string
}

// Service provides domain availability checking using the Namecheap API.
// It implements the DomainChecker interface for integration with MCP tools.
type Service struct {
	logger *zap.Logger
	config Config
}

// Config holds the configuration required to authenticate with the Namecheap API.
// All fields are required for successful API authentication.
type Config struct {
	// APIUser is the Namecheap API username
	APIUser  string
	// APIKey is the Namecheap API key for authentication
	APIKey   string
	// UserName is the Namecheap account username
	UserName string
	// ClientIP is the whitelisted IP address for API access
	ClientIP string
	// Endpoint is the Namecheap API endpoint URL (sandbox or production)
	Endpoint string
}

// ParamsIn represents the input parameters for domain availability checking.
// It contains the list of domains to be checked via the Namecheap API.
type ParamsIn struct {
	// Domains is the list of domain names to check for availability
	Domains []string `json:"domains" jsonschema:"The domains to check, e.g. example.com,example.org"`
}

// ParamsOut represents the output of domain availability checking.
// It contains the results for all domains that were checked.
type ParamsOut struct {
	// Results contains the availability information for each checked domain
	Results []Result `json:"results" jsonschema:"The results of the domain checks"`
}

// Result contains the availability and pricing information for a single domain.
// It includes availability status, premium domain information, and associated fees.
type Result struct {
	// Domain is the domain name that was checked
	Domain                   string  `json:"domain" jsonschema:"The domain that was checked"`
	// Available indicates if the domain is available for registration
	Available                bool    `json:"available" jsonschema:"Indicates if the domain is available for registration"`
	// IsPremiumName indicates whether the domain is classified as premium
	IsPremiumName            bool    `json:"isPremiumName" jsonschema:"Indicates whether the domain name is premium"`
	// PremiumRegistrationPrice is the registration cost for premium domains
	PremiumRegistrationPrice float64 `json:"premiumRegistrationPrice,omitempty" jsonschema:"Registration price"`
	// PremiumRenewalPrice is the annual renewal cost for premium domains
	PremiumRenewalPrice      float64 `json:"premiumRenewalPrice,omitempty" jsonschema:"Renewal price for premium domain"`
	// IcannFee is the ICANN registry fee associated with the domain
	IcannFee                 float64 `json:"icannFee,omitempty" jsonschema:"Fee charged by ICANN"`
	// EapFee is the Early Access Program fee for premium domains
	EapFee float64 `json:"eapFee,omitempty" jsonschema:"EAP fee"`
	// Error contains any error message if the domain check failed
	Error                    string  `json:"error,omitempty" jsonschema:"Error message if domain check failed"`
}

// APIResponse represents the XML response structure from the Namecheap API.
type APIResponse struct {
	XMLName         xml.Name        `xml:"ApiResponse"`
	Status          string          `xml:"Status,attr"`
	Errors          Errors          `xml:"Errors"`
	CommandResponse CommandResponse `xml:"CommandResponse"`
}

// Errors represents the errors section of the API response.
type Errors struct {
	Error []Error `xml:"Error"`
}

// Error represents individual error information from the API response.
type Error struct {
	Number  string `xml:"Number,attr"`
	Message string `xml:",chardata"`
}

// CommandResponse represents the command response section of the API response.
type CommandResponse struct {
	Type               string              `xml:"Type,attr"`
	DomainCheckResults []DomainCheckResult `xml:"DomainCheckResult"`
}

// DomainCheckResult represents individual domain check results from the API response.
type DomainCheckResult struct {
	Domain                   string `xml:"Domain,attr"`
	Available                string `xml:"Available,attr"`
	IsPremiumName            string `xml:"IsPremiumName,attr"`
	PremiumRegistrationPrice string `xml:"PremiumRegistrationPrice,attr"`
	PremiumRenewalPrice      string `xml:"PremiumRenewalPrice,attr"`
	IcannFee                 string `xml:"IcannFee,attr"`
	EapFee                   string `xml:"EapFee,attr"`
	ErrorNo                  string `xml:"ErrorNo,attr"`
	Description              string `xml:"Description,attr"`
}

// NewNamecheapTool creates a new NamecheapService with the provided logger and configuration.
// It validates that all required API credentials are present and returns an error if any are missing.
// The returned service implements the DomainChecker interface for checking domain availability.
func NewNamecheapTool(logger *zap.Logger, config Config) (*Service, error) {
	if config.APIUser == "" || config.APIKey == "" || config.UserName == "" || config.ClientIP == "" {
		return nil, ErrMissingAPICredentials
	}

	return &Service{
		logger: logger,
		config: config,
	}, nil
}

// Description returns a description of the Namecheap service.
func (n *Service) Description() string {
	return "Check domain availability using Namecheap API"
}

// Name returns the name of the Namecheap service.
func (n *Service) Name() string {
	return "check_availability_namecheap"
}

// DomainsCheck checks domain availability for the given list of domains using the Namecheap API.
// It accepts up to 50 domains in a single request and returns detailed availability information
// including premium domain pricing and associated fees. Returns ErrMissingDomains if no domains
// are provided, or an error if more than 50 domains are requested.
func (n *Service) DomainsCheck(domains []string) ([]Result, error) {
	if len(domains) == 0 {
		return nil, ErrMissingDomains
	}

	if len(domains) > maxDomainsPerCheck {
		return nil, ErrMaxDomainsExceeded
	}

	return n.checkDomains(domains)
}

func (n *Service) checkDomains(domains []string) ([]Result, error) {
	n.logger.Debug("Checking domains with Namecheap API",
		zap.Strings("domains", domains),
	)

	domainList := strings.Join(domains, ",")

	reqURL, err := n.buildRequestURL(n.config.Endpoint, domainList)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
	}

	n.logger.Debug("Making Namecheap API call",
		zap.String("url", reqURL),
		zap.Int("domain_count", len(domains)),
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{ //nolint:exhaustruct
		Timeout: time.Second * httpTimeoutSeconds,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var apiResp APIResponse

	decoder := xml.NewDecoder(resp.Body)

	err = decoder.Decode(&apiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	if apiResp.Status != "OK" {
		errorMsg := "unknown error"
		if len(apiResp.Errors.Error) > 0 {
			errorMsg = apiResp.Errors.Error[0].Message
		}

		return nil, fmt.Errorf("%w: %s", ErrAPIError, errorMsg)
	}

	results := n.parseResults(apiResp.CommandResponse.DomainCheckResults)

	n.logger.Debug("Domain check completed",
		zap.Int("domains_checked", len(results)),
		zap.Any("results", results),
	)

	return results, nil
}

func (n *Service) buildRequestURL(baseURL, domainList string) (string, error) {
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("ApiUser", n.config.APIUser)
	params.Add("ApiKey", n.config.APIKey)
	params.Add("UserName", n.config.UserName)
	params.Add("ClientIp", n.config.ClientIP)
	params.Add("Command", "namecheap.domains.check")
	params.Add("DomainList", domainList)

	baseURLParsed.RawQuery = params.Encode()

	return baseURLParsed.String(), nil
}

func (n *Service) parseResults(domainResults []DomainCheckResult) []Result {
	results := make([]Result, 0, len(domainResults))

	for _, domainResult := range domainResults {
		result := Result{
			Domain:                   domainResult.Domain,
			Available:                domainResult.Available == "true",
			IsPremiumName:            domainResult.IsPremiumName == "true",
			PremiumRegistrationPrice: 0,
			PremiumRenewalPrice:      0,
			IcannFee:                 0,
			EapFee:                   0,
			Error:                    "",
		}

		if domainResult.ErrorNo != "0" && domainResult.Description != "" {
			result.Error = domainResult.Description
		}

		if result.IsPremiumName {
			price, regErr := ParseFloat(domainResult.PremiumRegistrationPrice)
			if regErr == nil {
				result.PremiumRegistrationPrice = price
			}

			price, renErr := ParseFloat(domainResult.PremiumRenewalPrice)
			if renErr == nil {
				result.PremiumRenewalPrice = price
			}
		}

		fee, icannErr := ParseFloat(domainResult.IcannFee)
		if icannErr == nil {
			result.IcannFee = fee
		}

		fee, eapErr := ParseFloat(domainResult.EapFee)
		if eapErr == nil {
			result.EapFee = fee
		}

		results = append(results, result)
	}

	return results
}

// ParseFloat is a helper function to parse float values from string, exported for testing.
func ParseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}

	return strconv.ParseFloat(s, 64)
}
