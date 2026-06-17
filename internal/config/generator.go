package config

import (
	"fmt"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

// GeneratorConfig represents the configuration for the telemetry generator
type GeneratorConfig struct {
	Output  OutputConfig  `yaml:"output"`
	Traces  TracesConfig  `yaml:"traces"`
	Metrics MetricsConfig `yaml:"metrics"`
	Logs    LogsConfig    `yaml:"logs"`
}

// OutputConfig configures where generated telemetry is written
type OutputConfig struct {
	Directory string `yaml:"directory"`
	Prefix    string `yaml:"prefix"`
}

// TracesConfig configures trace generation
type TracesConfig struct {
	Count            int                    `yaml:"count"`
	Spans            SpansConfig            `yaml:"spans"`
	Services         ServicesConfig         `yaml:"services"`
	CustomAttributes CustomAttributesConfig `yaml:"custom_attributes"`
}

// SpansConfig configures span generation within traces
type SpansConfig struct {
	AvgPerTrace     int              `yaml:"avg_per_trace"`
	StdDev          int              `yaml:"std_dev"`
	HighSpanTraces  HighSpanTraces   `yaml:"high_span_traces"`
}

// HighSpanTraces configures generation of traces with very high span counts
type HighSpanTraces struct {
	Enabled   bool `yaml:"enabled"`
	Count     int  `yaml:"count"`
	SpanCount int  `yaml:"span_count"`
}

// ServicesConfig configures service topology
type ServicesConfig struct {
	Count                int               `yaml:"count"`
	Names                []string          `yaml:"names"`
	Ingress              IngressConfig     `yaml:"ingress"`
	Namespaces           []string          `yaml:"namespaces"`
	NamespaceAssignments map[string]string `yaml:"namespace_assignments"`

	// ResolvedNamespaces is the final service-name → namespace map used during
	// trace generation. Populated by ApplyDefaults; not unmarshaled from YAML.
	ResolvedNamespaces map[string]string `yaml:"-"`
}

// IngressConfig configures ingress service(s)
type IngressConfig struct {
	Single  bool   `yaml:"single"`
	Service string `yaml:"service"`
}

// CustomAttributesConfig configures custom attributes on spans
type CustomAttributesConfig struct {
	Count int `yaml:"count"`
}

// MetricsConfig configures metric generation
type MetricsConfig struct {
	MetricCount         int                      `yaml:"metric_count"`
	TimeSeriesPerMetric TimeSeriesPerMetricConfig `yaml:"timeseries_per_metric"`
}

// TimeSeriesPerMetricConfig defines the range of time series per metric
type TimeSeriesPerMetricConfig struct {
	Min     int `yaml:"min"`
	Max     int `yaml:"max"`
	Default int `yaml:"default"`
}

// LogsConfig configures log generation
type LogsConfig struct {
	Count int              `yaml:"count"`
	Types LogTypesConfig   `yaml:"types"`
}

// LogTypesConfig configures different types of logs
type LogTypesConfig struct {
	HTTPAccess  HTTPAccessConfig  `yaml:"http_access"`
	Application ApplicationConfig `yaml:"application"`
	System      SystemConfig      `yaml:"system"`
}

// HTTPAccessConfig configures HTTP access log generation
type HTTPAccessConfig struct {
	Percentage int `yaml:"percentage"`
}

// ApplicationConfig configures application log generation
type ApplicationConfig struct {
	Percentage int `yaml:"percentage"`
	Services   int `yaml:"services"`
}

// SystemConfig configures system log generation
type SystemConfig struct {
	Percentage int `yaml:"percentage"`
}

// LoadGeneratorConfig loads and validates a generator configuration from a file
func LoadGeneratorConfig(path string) (*GeneratorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GeneratorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply defaults
	config.ApplyDefaults()

	return &config, nil
}

// Memory estimation constants (approximate bytes per object in memory)
const (
	// Traces: Each span in memory is ~2KB (protobuf overhead, IDs, attributes, etc.)
	bytesPerSpan = 2048

	// Metrics: Each data point is ~400 bytes
	bytesPerMetricDataPoint = 400

	// Logs: Each log record is ~800 bytes
	bytesPerLogRecord = 800

	// Maximum memory usage for sender (10GB limit)
	maxMemoryBytes = 10 * 1024 * 1024 * 1024 // 10GB
)

// EstimateMemoryUsage calculates the approximate memory usage in bytes for the sender
func (c *GeneratorConfig) EstimateMemoryUsage() int64 {
	var totalBytes int64

	// Calculate trace span memory
	normalTraceSpans := c.Traces.Count * c.Traces.Spans.AvgPerTrace
	highSpanTraceSpans := 0
	if c.Traces.Spans.HighSpanTraces.Enabled {
		highSpanTraceSpans = c.Traces.Spans.HighSpanTraces.Count * c.Traces.Spans.HighSpanTraces.SpanCount
	}
	totalSpans := normalTraceSpans + highSpanTraceSpans
	totalBytes += int64(totalSpans) * bytesPerSpan

	// Calculate metric data point memory
	// Use average of min and max for estimation
	avgTimeSeries := (c.Metrics.TimeSeriesPerMetric.Min + c.Metrics.TimeSeriesPerMetric.Max) / 2
	if c.Metrics.TimeSeriesPerMetric.Default > 0 {
		avgTimeSeries = c.Metrics.TimeSeriesPerMetric.Default
	}
	totalDataPoints := c.Metrics.MetricCount * avgTimeSeries
	totalBytes += int64(totalDataPoints) * bytesPerMetricDataPoint

	// Calculate log record memory
	totalBytes += int64(c.Logs.Count) * bytesPerLogRecord

	return totalBytes
}

// Validate checks if the configuration is valid
func (c *GeneratorConfig) Validate() error {
	if c.Output.Directory == "" {
		return fmt.Errorf("output.directory is required")
	}

	// Check for negative counts
	if c.Traces.Count < 0 {
		return fmt.Errorf("traces.count must be non-negative")
	}

	if c.Metrics.MetricCount < 0 {
		return fmt.Errorf("metrics.metric_count must be non-negative")
	}

	if c.Logs.Count < 0 {
		return fmt.Errorf("logs.count must be non-negative")
	}

	// Ensure at least one telemetry type is enabled
	if c.Traces.Count == 0 && c.Metrics.MetricCount == 0 && c.Logs.Count == 0 {
		return fmt.Errorf("at least one telemetry type must be enabled (traces.count, metrics.metric_count, or logs.count must be > 0)")
	}

	// Only validate trace configuration if traces are enabled
	if c.Traces.Count > 0 {
		if c.Traces.Spans.AvgPerTrace < 1 {
			return fmt.Errorf("traces.spans.avg_per_trace must be at least 1")
		}

		if c.Traces.Spans.StdDev < 0 {
			return fmt.Errorf("traces.spans.std_dev must be non-negative")
		}

		if c.Traces.Services.Count < 1 {
			return fmt.Errorf("traces.services.count must be at least 1")
		}

		if len(c.Traces.Services.Names) > 0 && len(c.Traces.Services.Names) != c.Traces.Services.Count {
			return fmt.Errorf("traces.services.names length must match traces.services.count")
		}

		if err := c.validateNamespaceConfig(); err != nil {
			return err
		}
	}

	// Only validate metrics configuration if metrics are enabled
	if c.Metrics.MetricCount > 0 {
		if c.Metrics.MetricCount > 200 {
			return fmt.Errorf("metrics.metric_count must not exceed 200 (requested: %d)", c.Metrics.MetricCount)
		}

		if c.Metrics.TimeSeriesPerMetric.Min < 1 {
			return fmt.Errorf("metrics.timeseries_per_metric.min must be at least 1")
		}

		if c.Metrics.TimeSeriesPerMetric.Max < c.Metrics.TimeSeriesPerMetric.Min {
			return fmt.Errorf("metrics.timeseries_per_metric.max must be >= min")
		}
	}

	// Only validate log configuration if logs are enabled
	if c.Logs.Count > 0 {
		totalLogPercentage := c.Logs.Types.HTTPAccess.Percentage +
			c.Logs.Types.Application.Percentage +
			c.Logs.Types.System.Percentage
		if totalLogPercentage != 100 {
			return fmt.Errorf("log type percentages must sum to 100, got %d", totalLogPercentage)
		}
	}

	// Validate memory usage doesn't exceed 10GB limit
	estimatedMemory := c.EstimateMemoryUsage()
	if estimatedMemory > maxMemoryBytes {
		memoryGB := float64(estimatedMemory) / (1024 * 1024 * 1024)
		maxGB := float64(maxMemoryBytes) / (1024 * 1024 * 1024)
		return fmt.Errorf("estimated sender memory usage (%.2f GB) exceeds maximum (%.0f GB). "+
			"Reduce trace count, spans per trace, high-span traces, metrics, or logs. "+
			"See documentation for memory calculation details", memoryGB, maxGB)
	}

	return nil
}

// ApplyDefaults sets default values for optional fields
func (c *GeneratorConfig) ApplyDefaults() {
	if c.Output.Prefix == "" {
		c.Output.Prefix = "telemetry"
	}

	// Only apply metrics defaults if metrics are enabled
	if c.Metrics.MetricCount > 0 && c.Metrics.TimeSeriesPerMetric.Default == 0 {
		c.Metrics.TimeSeriesPerMetric.Default = 300
	}

	// Only apply trace defaults if traces are enabled
	if c.Traces.Count > 0 {
		// Generate service names if not provided
		if len(c.Traces.Services.Names) == 0 {
			c.Traces.Services.Names = make([]string, c.Traces.Services.Count)
			for i := 0; i < c.Traces.Services.Count; i++ {
				c.Traces.Services.Names[i] = fmt.Sprintf("service-%d", i+1)
			}
		}

		// Set default ingress if single ingress and service not specified
		if c.Traces.Services.Ingress.Single && c.Traces.Services.Ingress.Service == "" {
			c.Traces.Services.Ingress.Service = c.Traces.Services.Names[0]
		}

		c.resolveServiceNamespaces()
	}
}

// validateNamespaceConfig checks that namespace_assignments only reference
// known services, and (when a namespaces list is provided) only reference
// namespace values declared in it.
func (c *GeneratorConfig) validateNamespaceConfig() error {
	svc := c.Traces.Services
	if len(svc.NamespaceAssignments) == 0 {
		return nil
	}

	knownServices := make(map[string]struct{}, len(svc.Names))
	for _, name := range svc.Names {
		knownServices[name] = struct{}{}
	}

	knownNamespaces := make(map[string]struct{}, len(svc.Namespaces))
	for _, ns := range svc.Namespaces {
		knownNamespaces[ns] = struct{}{}
	}

	for service, namespace := range svc.NamespaceAssignments {
		if _, ok := knownServices[service]; !ok {
			return fmt.Errorf("traces.services.namespace_assignments references unknown service %q (must appear in traces.services.names)", service)
		}
		if len(svc.Namespaces) > 0 {
			if _, ok := knownNamespaces[namespace]; !ok {
				return fmt.Errorf("traces.services.namespace_assignments value %q for service %q must appear in traces.services.namespaces", namespace, service)
			}
		}
	}

	return nil
}

// resolveServiceNamespaces builds the final service-name → namespace map.
// Explicit assignments win; remaining services are distributed round-robin
// across the namespaces list in Names order so the same config always yields
// the same mapping.
func (c *GeneratorConfig) resolveServiceNamespaces() {
	svc := &c.Traces.Services
	if len(svc.Namespaces) == 0 && len(svc.NamespaceAssignments) == 0 {
		return
	}

	resolved := make(map[string]string, len(svc.Names))
	maps.Copy(resolved, svc.NamespaceAssignments)

	if len(svc.Namespaces) > 0 {
		unassignedIdx := 0
		for _, name := range svc.Names {
			if _, ok := resolved[name]; ok {
				continue
			}
			resolved[name] = svc.Namespaces[unassignedIdx%len(svc.Namespaces)]
			unassignedIdx++
		}
	}

	svc.ResolvedNamespaces = resolved
}
