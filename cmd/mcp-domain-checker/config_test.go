package main

import (
	"testing"
)

func TestCreateLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		logLevel  string
		logFormat string
	}{
		// Log levels
		{name: "debug level", logLevel: "debug", logFormat: "production"},
		{name: "debug level uppercase", logLevel: "DEBUG", logFormat: "production"},
		{name: "debug level mixed case", logLevel: "Debug", logFormat: "production"},
		{name: "info level", logLevel: "info", logFormat: "production"},
		{name: "warn level", logLevel: "warn", logFormat: "production"},
		{name: "warning level alias", logLevel: "warning", logFormat: "production"},
		{name: "error level", logLevel: "error", logFormat: "production"},
		{name: "fatal level", logLevel: "fatal", logFormat: "production"},
		{name: "panic level", logLevel: "panic", logFormat: "production"},
		{name: "invalid level defaults to info", logLevel: "invalid", logFormat: "production"},
		{name: "empty level defaults to info", logLevel: "", logFormat: "production"},
		// Log formats
		{name: "production format", logLevel: "info", logFormat: "production"},
		{name: "development format", logLevel: "info", logFormat: "development"},
		{name: "invalid format uses production", logLevel: "info", logFormat: "invalid"},
		{name: "empty format uses production", logLevel: "info", logFormat: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config{
				LogLevel:          tt.logLevel,
				LogFormat:         tt.logFormat,
				NamecheapAPIUser:  "",
				NamecheapAPIKey:   "",
				NamecheapUserName: "",
				NamecheapClientIP: "",
				NamecheapEndpoint: "",
			}

			logger, err := createLogger(cfg)
			if err != nil {
				t.Fatalf("createLogger() unexpected error: %v", err)
			}

			if logger == nil {
				t.Fatal("createLogger() returned nil logger")
			}

			_ = logger.Sync()
		})
	}
}
