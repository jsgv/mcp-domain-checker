package main

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type config struct {
	LogLevel          string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat         string `env:"LOG_FORMAT" envDefault:"production"`
	NamecheapAPIUser  string `env:"NAMECHEAP_API_USER"`
	NamecheapAPIKey   string `env:"NAMECHEAP_API_KEY"`
	NamecheapUserName string `env:"NAMECHEAP_USERNAME"`
	NamecheapClientIP string `env:"NAMECHEAP_CLIENT_IP"`
	NamecheapEndpoint string `env:"NAMECHEAP_ENDPOINT" envDefault:"https://api.namecheap.com/xml.response"`
}

// createLogger creates and configures a zap logger based on the provided configuration.
// It supports different log levels (debug, info, warn, error, fatal, panic) and formats (production, development).
// The logger defaults to info level and production format if invalid values are provided.
func createLogger(cfg *config) (*zap.Logger, error) {
	logLevel := strings.ToLower(cfg.LogLevel)

	var level zapcore.Level

	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	case "panic":
		level = zapcore.PanicLevel
	case "info":
		level = zapcore.InfoLevel
	default:
		level = zapcore.InfoLevel
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(level)

	if cfg.LogFormat == "development" {
		loggerConfig = zap.NewDevelopmentConfig()
		loggerConfig.Level = zap.NewAtomicLevelAt(level)
	}

	return loggerConfig.Build()
}
