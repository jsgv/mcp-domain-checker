package namecheap

import (
	"net/url"
	"testing"

	"go.uber.org/zap"
)

func TestParseFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "empty string returns zero",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid integer",
			input:   "100",
			want:    100,
			wantErr: false,
		},
		{
			name:    "valid float",
			input:   "10.5",
			want:    10.5,
			wantErr: false,
		},
		{
			name:    "valid small float",
			input:   "0.01",
			want:    0.01,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative number",
			input:   "-10.5",
			want:    -10.5,
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "mixed invalid",
			input:   "10.5abc",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()

	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "127.0.0.1",
				Endpoint: "https://api.namecheap.com/xml.response",
			},
			wantErr: nil,
		},
		{
			name: "missing APIUser",
			config: Config{
				APIUser:  "",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "127.0.0.1",
			},
			wantErr: ErrMissingAPICredentials,
		},
		{
			name: "missing APIKey",
			config: Config{
				APIUser:  "user",
				APIKey:   "",
				UserName: "username",
				ClientIP: "127.0.0.1",
			},
			wantErr: ErrMissingAPICredentials,
		},
		{
			name: "missing UserName",
			config: Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "",
				ClientIP: "127.0.0.1",
			},
			wantErr: ErrMissingAPICredentials,
		},
		{
			name: "missing ClientIP",
			config: Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "",
			},
			wantErr: ErrMissingAPICredentials,
		},
		{
			name: "all fields missing",
			config: Config{
				APIUser:  "",
				APIKey:   "",
				UserName: "",
				ClientIP: "",
			},
			wantErr: ErrMissingAPICredentials,
		},
		{
			name: "endpoint can be empty",
			config: Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "127.0.0.1",
				Endpoint: "",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, err := NewService(logger, tt.config)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				}

				if service != nil {
					t.Error("NewService() returned service when error expected")
				}
			} else {
				if err != nil {
					t.Errorf("NewService() unexpected error = %v", err)
				}

				if service == nil {
					t.Error("NewService() returned nil service")
				}
			}
		})
	}
}

func TestDomainsCheck_Validation(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	config := Config{
		APIUser:  "user",
		APIKey:   "key",
		UserName: "username",
		ClientIP: "127.0.0.1",
		Endpoint: "https://api.namecheap.com/xml.response",
	}

	service, err := NewService(logger, config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name    string
		domains []string
		wantErr error
	}{
		{
			name:    "empty domains",
			domains: []string{},
			wantErr: ErrMissingDomains,
		},
		{
			name:    "nil domains",
			domains: nil,
			wantErr: ErrMissingDomains,
		},
		{
			name:    "51 domains exceeds limit",
			domains: make([]string, 51),
			wantErr: ErrMaxDomainsExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.DomainsCheck(tt.domains)
			if err != tt.wantErr {
				t.Errorf("DomainsCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildRequestURL(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	config := Config{
		APIUser:  "testuser",
		APIKey:   "testkey",
		UserName: "testusername",
		ClientIP: "192.168.1.1",
		Endpoint: "https://api.namecheap.com/xml.response",
	}

	service, err := NewService(logger, config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name       string
		baseURL    string
		domainList string
		wantErr    bool
		checkURL   func(t *testing.T, resultURL string)
	}{
		{
			name:       "valid URL with single domain",
			baseURL:    "https://api.namecheap.com/xml.response",
			domainList: "example.com",
			wantErr:    false,
			checkURL: func(t *testing.T, resultURL string) {
				t.Helper()

				parsed, err := url.Parse(resultURL)
				if err != nil {
					t.Fatalf("Failed to parse result URL: %v", err)
				}

				query := parsed.Query()
				if query.Get("ApiUser") != "testuser" {
					t.Errorf("ApiUser = %v, want testuser", query.Get("ApiUser"))
				}

				if query.Get("ApiKey") != "testkey" {
					t.Errorf("ApiKey = %v, want testkey", query.Get("ApiKey"))
				}

				if query.Get("UserName") != "testusername" {
					t.Errorf("UserName = %v, want testusername", query.Get("UserName"))
				}

				if query.Get("ClientIp") != "192.168.1.1" {
					t.Errorf("ClientIp = %v, want 192.168.1.1", query.Get("ClientIp"))
				}

				if query.Get("Command") != "namecheap.domains.check" {
					t.Errorf("Command = %v, want namecheap.domains.check", query.Get("Command"))
				}

				if query.Get("DomainList") != "example.com" {
					t.Errorf("DomainList = %v, want example.com", query.Get("DomainList"))
				}
			},
		},
		{
			name:       "valid URL with multiple domains",
			baseURL:    "https://api.namecheap.com/xml.response",
			domainList: "example.com,example.org,example.net",
			wantErr:    false,
			checkURL: func(t *testing.T, resultURL string) {
				t.Helper()

				parsed, err := url.Parse(resultURL)
				if err != nil {
					t.Fatalf("Failed to parse result URL: %v", err)
				}

				query := parsed.Query()
				if query.Get("DomainList") != "example.com,example.org,example.net" {
					t.Errorf("DomainList = %v, want example.com,example.org,example.net", query.Get("DomainList"))
				}
			},
		},
		{
			name:       "invalid base URL",
			baseURL:    "://invalid",
			domainList: "example.com",
			wantErr:    true,
			checkURL:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resultURL, err := service.buildRequestURL(tt.baseURL, tt.domainList)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildRequestURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkURL != nil {
				tt.checkURL(t, resultURL)
			}
		})
	}
}

func TestParseResults(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	config := Config{
		APIUser:  "user",
		APIKey:   "key",
		UserName: "username",
		ClientIP: "127.0.0.1",
	}

	service, err := NewService(logger, config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name  string
		input []DomainCheckResult
		want  []Result
	}{
		{
			name:  "empty input",
			input: []DomainCheckResult{},
			want:  []Result{},
		},
		{
			name: "available domain",
			input: []DomainCheckResult{
				{
					Domain:    "available.com",
					Available: "true",
					ErrorNo:   "0",
				},
			},
			want: []Result{
				{
					Domain:    "available.com",
					Available: true,
				},
			},
		},
		{
			name: "unavailable domain",
			input: []DomainCheckResult{
				{
					Domain:    "taken.com",
					Available: "false",
					ErrorNo:   "0",
				},
			},
			want: []Result{
				{
					Domain:    "taken.com",
					Available: false,
				},
			},
		},
		{
			name: "premium domain with pricing",
			input: []DomainCheckResult{
				{
					Domain:                   "premium.com",
					Available:                "true",
					IsPremiumName:            "true",
					PremiumRegistrationPrice: "1000.00",
					PremiumRenewalPrice:      "500.00",
					ErrorNo:                  "0",
				},
			},
			want: []Result{
				{
					Domain:                   "premium.com",
					Available:                true,
					IsPremiumName:            true,
					PremiumRegistrationPrice: 1000.00,
					PremiumRenewalPrice:      500.00,
				},
			},
		},
		{
			name: "domain with ICANN and EAP fees",
			input: []DomainCheckResult{
				{
					Domain:    "example.com",
					Available: "true",
					IcannFee:  "0.18",
					EapFee:    "10.00",
					ErrorNo:   "0",
				},
			},
			want: []Result{
				{
					Domain:    "example.com",
					Available: true,
					IcannFee:  0.18,
					EapFee:    10.00,
				},
			},
		},
		{
			name: "domain with error",
			input: []DomainCheckResult{
				{
					Domain:      "error.com",
					Available:   "false",
					ErrorNo:     "1",
					Description: "Domain check failed",
				},
			},
			want: []Result{
				{
					Domain:    "error.com",
					Available: false,
					Error:     "Domain check failed",
				},
			},
		},
		{
			name: "multiple domains mixed states",
			input: []DomainCheckResult{
				{
					Domain:    "available.com",
					Available: "true",
					ErrorNo:   "0",
				},
				{
					Domain:    "taken.com",
					Available: "false",
					ErrorNo:   "0",
				},
				{
					Domain:        "premium.io",
					Available:     "true",
					IsPremiumName: "true",
					ErrorNo:       "0",
				},
			},
			want: []Result{
				{
					Domain:    "available.com",
					Available: true,
				},
				{
					Domain:    "taken.com",
					Available: false,
				},
				{
					Domain:        "premium.io",
					Available:     true,
					IsPremiumName: true,
				},
			},
		},
		{
			name: "non-premium domain ignores premium pricing",
			input: []DomainCheckResult{
				{
					Domain:                   "regular.com",
					Available:                "true",
					IsPremiumName:            "false",
					PremiumRegistrationPrice: "1000.00",
					PremiumRenewalPrice:      "500.00",
					ErrorNo:                  "0",
				},
			},
			want: []Result{
				{
					Domain:                   "regular.com",
					Available:                true,
					IsPremiumName:            false,
					PremiumRegistrationPrice: 0,
					PremiumRenewalPrice:      0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := service.parseResults(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("parseResults() returned %d results, want %d", len(got), len(tt.want))
			}

			for i, want := range tt.want {
				if got[i].Domain != want.Domain {
					t.Errorf("result[%d].Domain = %v, want %v", i, got[i].Domain, want.Domain)
				}

				if got[i].Available != want.Available {
					t.Errorf("result[%d].Available = %v, want %v", i, got[i].Available, want.Available)
				}

				if got[i].IsPremiumName != want.IsPremiumName {
					t.Errorf("result[%d].IsPremiumName = %v, want %v", i, got[i].IsPremiumName, want.IsPremiumName)
				}

				if got[i].PremiumRegistrationPrice != want.PremiumRegistrationPrice {
					t.Errorf("result[%d].PremiumRegistrationPrice = %v, want %v", i, got[i].PremiumRegistrationPrice, want.PremiumRegistrationPrice)
				}

				if got[i].PremiumRenewalPrice != want.PremiumRenewalPrice {
					t.Errorf("result[%d].PremiumRenewalPrice = %v, want %v", i, got[i].PremiumRenewalPrice, want.PremiumRenewalPrice)
				}

				if got[i].IcannFee != want.IcannFee {
					t.Errorf("result[%d].IcannFee = %v, want %v", i, got[i].IcannFee, want.IcannFee)
				}

				if got[i].EapFee != want.EapFee {
					t.Errorf("result[%d].EapFee = %v, want %v", i, got[i].EapFee, want.EapFee)
				}

				if got[i].Error != want.Error {
					t.Errorf("result[%d].Error = %v, want %v", i, got[i].Error, want.Error)
				}
			}
		})
	}
}
