package namecheap_test

import (
	"errors"
	"testing"

	"github.com/jsgv/mcp-domain-checker/internal/pkg/namecheap"
	"go.uber.org/zap"
)

//nolint:funlen
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

			got, err := namecheap.ParseFloat(tt.input)
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

//nolint:funlen
func TestNewService(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()

	tests := []struct {
		name    string
		config  namecheap.Config
		wantErr error
	}{
		{
			name: "valid config",
			config: namecheap.Config{
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
			config: namecheap.Config{
				APIUser:  "",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "127.0.0.1",
				Endpoint: "",
			},
			wantErr: namecheap.ErrMissingAPICredentials,
		},
		{
			name: "missing APIKey",
			config: namecheap.Config{
				APIUser:  "user",
				APIKey:   "",
				UserName: "username",
				ClientIP: "127.0.0.1",
				Endpoint: "",
			},
			wantErr: namecheap.ErrMissingAPICredentials,
		},
		{
			name: "missing UserName",
			config: namecheap.Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "",
				ClientIP: "127.0.0.1",
				Endpoint: "",
			},
			wantErr: namecheap.ErrMissingAPICredentials,
		},
		{
			name: "missing ClientIP",
			config: namecheap.Config{
				APIUser:  "user",
				APIKey:   "key",
				UserName: "username",
				ClientIP: "",
				Endpoint: "",
			},
			wantErr: namecheap.ErrMissingAPICredentials,
		},
		{
			name: "all fields missing",
			config: namecheap.Config{
				APIUser:  "",
				APIKey:   "",
				UserName: "",
				ClientIP: "",
				Endpoint: "",
			},
			wantErr: namecheap.ErrMissingAPICredentials,
		},
		{
			name: "endpoint can be empty",
			config: namecheap.Config{
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

			service, err := namecheap.NewService(logger, tt.config)

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr != nil && service != nil {
				t.Error("NewService() returned service when error expected")
			}

			if tt.wantErr == nil && err != nil {
				t.Errorf("NewService() unexpected error = %v", err)
			}

			if tt.wantErr == nil && service == nil {
				t.Error("NewService() returned nil service")
			}
		})
	}
}

func TestDomainsCheck_Validation(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	config := namecheap.Config{
		APIUser:  "user",
		APIKey:   "key",
		UserName: "username",
		ClientIP: "127.0.0.1",
		Endpoint: "https://api.namecheap.com/xml.response",
	}

	service, err := namecheap.NewService(logger, config)
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
			wantErr: namecheap.ErrMissingDomains,
		},
		{
			name:    "nil domains",
			domains: nil,
			wantErr: namecheap.ErrMissingDomains,
		},
		{
			name:    "51 domains exceeds limit",
			domains: make([]string, 51),
			wantErr: namecheap.ErrMaxDomainsExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.DomainsCheck(tt.domains)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DomainsCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
