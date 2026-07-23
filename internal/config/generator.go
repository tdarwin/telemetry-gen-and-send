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
	Limits  LimitsConfig  `yaml:"limits"`
}

// LimitsConfig overrides the safety cap on estimated in-memory dataset size.
// It exists for the Refinery-stress scenarios where fat spans and/or gigatraces
// intentionally produce very large datasets. Both fields default to the historic
// behavior (a 10GB cap) so existing configs are unaffected.
type LimitsConfig struct {
	// MaxMemoryGB is the estimated-memory ceiling enforced by Validate().
	// 0 => default of 10 (applied in ApplyDefaults).
	MaxMemoryGB float64 `yaml:"max_memory_gb"`
	// AllowUnbounded disables the memory cap entirely. Intended only for
	// deliberate stress runs; the generator prints a warning when it is set.
	AllowUnbounded bool `yaml:"allow_unbounded"`
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
	Root             RootConfig             `yaml:"root"`
}

// RootConfig controls "missing/late root" trace shapes used to stress a
// sampling proxy's trace cache (e.g. Refinery). Both sub-features default OFF,
// so omitting this section preserves the historic behavior where every trace
// has a normal root span that arrives with the rest of the trace.
type RootConfig struct {
	Rootless RootlessConfig `yaml:"rootless"`
	LateRoot LateRootConfig `yaml:"late_root"`
}

// RootlessConfig gives a percentage of traces a phantom (never-emitted) parent
// on their root span, so the receiver never sees a root and holds the trace
// until its trace timeout expires. Span count is unchanged.
type RootlessConfig struct {
	Enabled    bool `yaml:"enabled"`
	Percentage int  `yaml:"percentage"`
}

// LateRootConfig stamps the root span of a percentage of traces with an emit
// delay (via the _template.emit_delay_ms attribute) so the sender exports the
// root later than the rest of the trace — after the receiver's trace timeout.
type LateRootConfig struct {
	Enabled    bool `yaml:"enabled"`
	Percentage int  `yaml:"percentage"`
	DelayMs    int  `yaml:"delay_ms"`
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
	// Single selects one named entry point (Service). When false, every
	// service is a possible trace entry point.
	Single  bool   `yaml:"single"`
	Service string `yaml:"service"`
}

// CustomAttributesConfig configures custom attributes on spans.
//
// The legacy behavior (Count only) attaches a small, random set of custom
// attributes to ~30% of spans, drawn from a pool of Count distinct schemas.
//
// The "fat span" fields below force a deterministic number of large
// attributes onto EVERY span, letting a config inflate per-span byte size
// (tens of KB) without changing span or event counts. They default to zero,
// which preserves the legacy behavior exactly.
type CustomAttributesConfig struct {
	// Count is the size of the legacy random-attribute schema pool.
	Count int `yaml:"count"`

	// PerSpanMin and PerSpanMax force a deterministic count of fat attributes
	// on every span (a random value in [min, max]). When PerSpanMax > 0 the
	// legacy 30%/1-3 random path is bypassed for fat attributes.
	PerSpanMin int `yaml:"per_span_min"`
	PerSpanMax int `yaml:"per_span_max"`

	// ValueBytes is the length (in bytes) of each fat attribute's string value.
	ValueBytes int `yaml:"value_bytes"`

	// KeyPrefix names the fat attributes: <prefix>.0 .. <prefix>.N-1. Distinct
	// keys create per-span column pressure in addition to raw byte size.
	// Defaults to "custom.fat" when fat mode is enabled.
	KeyPrefix string `yaml:"key_prefix"`
}

// FatSpansEnabled reports whether deterministic fat attributes should be
// attached to every span.
func (c CustomAttributesConfig) FatSpansEnabled() bool {
	return c.PerSpanMax > 0
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

	if config.Limits.AllowUnbounded {
		fmt.Println("WARNING: limits.allow_unbounded is set — the dataset memory cap is disabled. " +
			"Generation may consume very large amounts of memory and disk.")
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

	// Default maximum memory usage for the dataset (10GB). Overridable via
	// limits.max_memory_gb / limits.allow_unbounded.
	defaultMaxMemoryGB = 10

	// Per-fat-attribute overhead beyond the value bytes: key string plus
	// protobuf field/length framing. Deliberately conservative so the estimate
	// never under-counts a fat-span dataset.
	fatAttrKeyOverhead = 32
)

// perSpanBytes returns the estimated in-memory size of a single span, including
// any configured fat custom attributes. bytesPerSpan is the base cost (IDs,
// name, status, semantic attributes, protobuf overhead); fat attributes are
// added on top using the worst-case per-span count so validation can't
// under-estimate.
func (c *GeneratorConfig) perSpanBytes() int64 {
	base := int64(bytesPerSpan)
	ca := c.Traces.CustomAttributes
	if ca.FatSpansEnabled() {
		base += int64(ca.PerSpanMax) * int64(ca.ValueBytes+fatAttrKeyOverhead)
	}
	return base
}

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
	totalBytes += int64(totalSpans) * c.perSpanBytes()

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

		if err := c.validateCustomAttributes(); err != nil {
			return err
		}

		if err := c.validateRootConfig(); err != nil {
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

	// Validate estimated memory usage doesn't exceed the configured cap.
	// limits.allow_unbounded disables the check entirely (for deliberate
	// stress runs). limits.max_memory_gb (default 10) sets the ceiling.
	if !c.Limits.AllowUnbounded {
		maxGB := c.Limits.MaxMemoryGB
		if maxGB == 0 {
			maxGB = defaultMaxMemoryGB
		}
		maxBytes := int64(maxGB * 1024 * 1024 * 1024)
		estimatedMemory := c.EstimateMemoryUsage()
		if estimatedMemory > maxBytes {
			memoryGB := float64(estimatedMemory) / (1024 * 1024 * 1024)
			return fmt.Errorf("estimated sender memory usage (%.2f GB) exceeds maximum (%.2f GB). "+
				"Reduce trace count, spans per trace, high-span traces, custom attribute size, "+
				"metrics, or logs — or raise limits.max_memory_gb / set limits.allow_unbounded. "+
				"See documentation for memory calculation details", memoryGB, maxGB)
		}
	}

	return nil
}

// validateCustomAttributes checks the fat-span controls in
// traces.custom_attributes.
func (c *GeneratorConfig) validateCustomAttributes() error {
	ca := c.Traces.CustomAttributes
	if ca.PerSpanMin < 0 || ca.PerSpanMax < 0 {
		return fmt.Errorf("traces.custom_attributes.per_span_min/per_span_max must be non-negative")
	}
	if ca.PerSpanMax < ca.PerSpanMin {
		return fmt.Errorf("traces.custom_attributes.per_span_max must be >= per_span_min")
	}
	if ca.ValueBytes < 0 {
		return fmt.Errorf("traces.custom_attributes.value_bytes must be non-negative")
	}
	if ca.PerSpanMax > 0 && ca.ValueBytes == 0 {
		return fmt.Errorf("traces.custom_attributes.value_bytes must be > 0 when per_span_max > 0")
	}
	return nil
}

// validateRootConfig checks the missing/late root controls in traces.root.
func (c *GeneratorConfig) validateRootConfig() error {
	r := c.Traces.Root
	for _, p := range []struct {
		name string
		val  int
	}{
		{"rootless.percentage", r.Rootless.Percentage},
		{"late_root.percentage", r.LateRoot.Percentage},
	} {
		if p.val < 0 || p.val > 100 {
			return fmt.Errorf("traces.root.%s must be between 0 and 100", p.name)
		}
	}
	if r.LateRoot.Enabled && r.LateRoot.DelayMs <= 0 {
		return fmt.Errorf("traces.root.late_root.delay_ms must be > 0 when late_root is enabled")
	}
	return nil
}

// ApplyDefaults sets default values for optional fields
func (c *GeneratorConfig) ApplyDefaults() {
	if c.Output.Prefix == "" {
		c.Output.Prefix = "telemetry"
	}

	if c.Limits.MaxMemoryGB == 0 {
		c.Limits.MaxMemoryGB = defaultMaxMemoryGB
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

		// Default the fat-attribute key prefix when fat spans are enabled.
		if c.Traces.CustomAttributes.FatSpansEnabled() && c.Traces.CustomAttributes.KeyPrefix == "" {
			c.Traces.CustomAttributes.KeyPrefix = "custom.fat"
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
