package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceCheck represents a single service to be checked
type ServiceCheck struct {
	Name        string        `yaml:"name"`
	Type        string        `yaml:"type"`  // "http", "tcp", etc
	Target      string        `yaml:"target"`
	Interval    time.Duration `yaml:"interval"`
	Timeout     time.Duration `yaml:"timeout"`
	ExpectedStatus int        `yaml:"expectedStatus,omitempty"` // For HTTP checks
}

// Config represents the overall configuration
type Config struct {
	Checks    []ServiceCheck `yaml:"checks"`
	LogLevel  string         `yaml:"logLevel"`
	APIPort   int            `yaml:"apiPort"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set defaults if not specified
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.APIPort == 0 {
		config.APIPort = 8080
	}

	// Convert string durations to time.Duration
	for i := range config.Checks {
		if config.Checks[i].Interval <= 0 {
			config.Checks[i].Interval = 60 * time.Second
		}
		if config.Checks[i].Timeout <= 0 {
			config.Checks[i].Timeout = 5 * time.Second
		}
	}

	return &config, nil
}