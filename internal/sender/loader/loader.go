package loader

import (
	"fmt"
	"os"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpcollectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"
)

// Templates holds all loaded telemetry templates
type Templates struct {
	Traces  *otlpcollectortrace.ExportTraceServiceRequest
	Metrics *otlpcollectormetrics.ExportMetricsServiceRequest
	Logs    *otlpcollectorlogs.ExportLogsServiceRequest
}

// Loader handles loading telemetry templates from disk
type Loader struct{}

// NewLoader creates a new template loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads all configured templates
func (l *Loader) Load(tracesPath, metricsPath, logsPath string) (*Templates, error) {
	templates := &Templates{}

	// Load traces if path provided
	if tracesPath != "" {
		fmt.Printf("Loading traces from %s...\n", tracesPath)
		traces, err := l.loadTraces(tracesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load traces: %w", err)
		}
		templates.Traces = traces

		// Count spans
		spanCount := 0
		for _, rs := range traces.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				spanCount += len(ss.Spans)
			}
		}
		fmt.Printf("  Loaded %d resource spans with %d total spans\n", len(traces.ResourceSpans), spanCount)
	}

	// Load metrics if path provided
	if metricsPath != "" {
		fmt.Printf("Loading metrics from %s...\n", metricsPath)
		metrics, err := l.loadMetrics(metricsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load metrics: %w", err)
		}
		templates.Metrics = metrics

		// Count data points
		dataPoints := 0
		for _, rm := range metrics.ResourceMetrics {
			for _, sm := range rm.ScopeMetrics {
				for _, metric := range sm.Metrics {
					dataPoints += l.countMetricDataPoints(metric)
				}
			}
		}
		fmt.Printf("  Loaded %d metrics with %d data points\n",
			len(metrics.ResourceMetrics[0].ScopeMetrics[0].Metrics), dataPoints)
	}

	// Load logs if path provided
	if logsPath != "" {
		fmt.Printf("Loading logs from %s...\n", logsPath)
		logs, err := l.loadLogs(logsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load logs: %w", err)
		}
		templates.Logs = logs

		// Count log records
		logCount := 0
		for _, rl := range logs.ResourceLogs {
			for _, sl := range rl.ScopeLogs {
				logCount += len(sl.LogRecords)
			}
		}
		fmt.Printf("  Loaded %d log records\n", logCount)
	}

	return templates, nil
}

// loadTraces loads trace templates from a protobuf file
func (l *Loader) loadTraces(path string) (*otlpcollectortrace.ExportTraceServiceRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	request := &otlpcollectortrace.ExportTraceServiceRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return request, nil
}

// loadMetrics loads metric templates from a protobuf file
func (l *Loader) loadMetrics(path string) (*otlpcollectormetrics.ExportMetricsServiceRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	request := &otlpcollectormetrics.ExportMetricsServiceRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return request, nil
}

// loadLogs loads log templates from a protobuf file
func (l *Loader) loadLogs(path string) (*otlpcollectorlogs.ExportLogsServiceRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	request := &otlpcollectorlogs.ExportLogsServiceRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return request, nil
}

// countMetricDataPoints counts the number of data points in a metric
func (l *Loader) countMetricDataPoints(metric *otlpmetrics.Metric) int {
	switch data := metric.Data.(type) {
	case *otlpmetrics.Metric_Gauge:
		return len(data.Gauge.DataPoints)
	case *otlpmetrics.Metric_Sum:
		return len(data.Sum.DataPoints)
	case *otlpmetrics.Metric_Histogram:
		return len(data.Histogram.DataPoints)
	case *otlpmetrics.Metric_Summary:
		return len(data.Summary.DataPoints)
	default:
		return 0
	}
}
