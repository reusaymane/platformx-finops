package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	PrometheusURL   string
	AWSRegion       string
	Port            string
	CollectInterval time.Duration
	FakeMode        bool
	LogLevel        string
}

func Load() (*Config, error) {
	interval, err := time.ParseDuration(getEnv("COLLECT_INTERVAL", "60s"))
	if err != nil {
		return nil, fmt.Errorf("invalid COLLECT_INTERVAL: %w", err)
	}

	cfg := &Config{
		DatabaseURL:     requireEnv("DATABASE_URL"),
		PrometheusURL:   getEnv("PROMETHEUS_URL", "http://localhost:9090"),
		AWSRegion:       getEnv("AWS_REGION", "eu-west-3"),
		Port:            getEnv("PORT", "8080"),
		CollectInterval: interval,
		FakeMode:        getEnv("FAKE_MODE", "false") == "true",
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}
	return cfg, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
