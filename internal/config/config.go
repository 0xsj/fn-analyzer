// internal/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DSN              string        `json:"dsn"`              // Database connection string
	QueriesFile      string        `json:"queriesFile"`      // Path to critical queries JSON file
	OutputDir        string        `json:"outputDir"`        // Directory to save results
	Iterations       int           `json:"iterations"`       // Number of iterations per query
	Concurrency      int           `json:"concurrency"`      // Maximum concurrent queries
	WarmupIterations int           `json:"warmupIterations"` // Warmup iterations to stabilize connection pool
	Label            string        `json:"label"`            // Test run label (e.g., "before" or "after")
	Timeout          time.Duration `json:"timeoutSeconds"`   // Query timeout in seconds
	Verbose          bool          `json:"verbose"`          // Verbose output
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{
		DSN:              "root:password@tcp(localhost:3306)/database",
		OutputDir:        "./performance-results",
		Iterations:       50,
		Concurrency:      5,
		WarmupIterations: 100,
		Label:            "baseline",
		Timeout:          30 * time.Second,
		Verbose:          false,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("couldn't create config directory: %w", err)
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error creating default config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, fmt.Errorf("error writing default config: %w", err)
		}

		fmt.Printf("Created default config file at %s\n", path)
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.Iterations <= 0 {
		config.Iterations = 50
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}
	if config.WarmupIterations < 0 {
		config.WarmupIterations = 100
	}

	return config, nil
}
