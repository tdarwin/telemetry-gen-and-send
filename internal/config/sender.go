package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SenderConfig represents the configuration for the telemetry sender
type SenderConfig struct {
	Input      InputConfig      `yaml:"input"`
	OTLP       OTLPConfig       `yaml:"otlp"`
	Sending    SendingConfig    `yaml:"sending"`
	Timestamps TimestampsConfig `yaml:"timestamps"`
}

// InputConfig configures where to load telemetry templates from
type InputConfig struct {
	Traces  string `yaml:"traces"`
	Metrics string `yaml:"metrics"`
	Logs    string `yaml:"logs"`
}

// OTLPConfig configures the OTLP endpoint
type OTLPConfig struct {
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
	Insecure bool              `yaml:"insecure"`
}

// SendingConfig configures how telemetry is sent
type SendingConfig struct {
	RateLimit   RateLimitConfig   `yaml:"rate_limit"`
	BatchSize   BatchSizeConfig   `yaml:"batch_size"`
	Concurrency int               `yaml:"concurrency"`
	Duration    string            `yaml:"duration"`
	Multiplier  int               `yaml:"multiplier"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	EventsPerSecond int `yaml:"events_per_second"`
}

// BatchSizeConfig configures batch sizes for different signal types
type BatchSizeConfig struct {
	Traces  int `yaml:"traces"`
	Metrics int `yaml:"metrics"`
	Logs    int `yaml:"logs"`
}

// TimestampsConfig configures timestamp behavior
type TimestampsConfig struct {
	JitterMs   int `yaml:"jitter_ms"`
	BackdateMs int `yaml:"backdate_ms"`
}

// LoadSenderConfig loads and validates a sender configuration from a file
func LoadSenderConfig(path string) (*SenderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SenderConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand environment variables in config
	config.expandEnvVars()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply defaults
	config.ApplyDefaults()

	return &config, nil
}

// expandEnvVars expands environment variables in string fields
func (c *SenderConfig) expandEnvVars() {
	c.OTLP.Endpoint = os.ExpandEnv(c.OTLP.Endpoint)
	for k, v := range c.OTLP.Headers {
		c.OTLP.Headers[k] = os.ExpandEnv(v)
	}
	c.Input.Traces = os.ExpandEnv(c.Input.Traces)
	c.Input.Metrics = os.ExpandEnv(c.Input.Metrics)
	c.Input.Logs = os.ExpandEnv(c.Input.Logs)
}

// Validate checks if the configuration is valid
func (c *SenderConfig) Validate() error {
	// At least one input must be specified
	if c.Input.Traces == "" && c.Input.Metrics == "" && c.Input.Logs == "" {
		return fmt.Errorf("at least one input file (traces, metrics, or logs) must be specified")
	}

	if c.OTLP.Endpoint == "" {
		return fmt.Errorf("otlp.endpoint is required")
	}

	if c.Sending.RateLimit.EventsPerSecond <= 0 {
		return fmt.Errorf("sending.rate_limit.events_per_second must be positive")
	}

	if c.Sending.Concurrency <= 0 {
		return fmt.Errorf("sending.concurrency must be positive")
	}

	if c.Sending.Multiplier < 0 {
		return fmt.Errorf("sending.multiplier must be non-negative (0 = infinite)")
	}

	// Validate duration format if specified
	if c.Sending.Duration != "" && c.Sending.Duration != "0" {
		if _, err := time.ParseDuration(c.Sending.Duration); err != nil {
			return fmt.Errorf("invalid sending.duration format: %w", err)
		}
	}

	if c.Timestamps.JitterMs < 0 {
		return fmt.Errorf("timestamps.jitter_ms must be non-negative")
	}

	if c.Timestamps.BackdateMs < 0 {
		return fmt.Errorf("timestamps.backdate_ms must be non-negative")
	}

	return nil
}

// ApplyDefaults sets default values for optional fields
func (c *SenderConfig) ApplyDefaults() {
	if c.Sending.BatchSize.Traces == 0 {
		c.Sending.BatchSize.Traces = 100
	}
	if c.Sending.BatchSize.Metrics == 0 {
		c.Sending.BatchSize.Metrics = 1000
	}
	if c.Sending.BatchSize.Logs == 0 {
		c.Sending.BatchSize.Logs = 1000
	}

	if c.Sending.Duration == "" {
		c.Sending.Duration = "0" // infinite
	}

	if c.Timestamps.JitterMs == 0 {
		c.Timestamps.JitterMs = 1000 // 1 second default
	}
}

// GetDuration parses and returns the sending duration
func (c *SenderConfig) GetDuration() (time.Duration, error) {
	if c.Sending.Duration == "0" {
		return 0, nil // infinite
	}
	return time.ParseDuration(c.Sending.Duration)
}

// HasTraces returns true if traces input is configured
func (c *SenderConfig) HasTraces() bool {
	return c.Input.Traces != "" && !strings.HasSuffix(c.Input.Traces, "null")
}

// HasMetrics returns true if metrics input is configured
func (c *SenderConfig) HasMetrics() bool {
	return c.Input.Metrics != "" && !strings.HasSuffix(c.Input.Metrics, "null")
}

// HasLogs returns true if logs input is configured
func (c *SenderConfig) HasLogs() bool {
	return c.Input.Logs != "" && !strings.HasSuffix(c.Input.Logs, "null")
}
