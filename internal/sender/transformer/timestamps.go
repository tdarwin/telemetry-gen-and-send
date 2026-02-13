package transformer

import (
	"math/rand"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	otlplogs "go.opentelemetry.io/proto/otlp/logs/v1"
)

// TimestampInjector adds timestamps to telemetry
type TimestampInjector struct {
	jitterMs   int
	backdateMs int
}

// NewTimestampInjector creates a new timestamp injector
func NewTimestampInjector(jitterMs, backdateMs int) *TimestampInjector {
	return &TimestampInjector{
		jitterMs:   jitterMs,
		backdateMs: backdateMs,
	}
}

// InjectSpanTimestamps adds timestamps to spans while preserving relative timing
func (t *TimestampInjector) InjectSpanTimestamps(spans []*otlptrace.Span) {
	if len(spans) == 0 {
		return
	}

	// Get current time with optional backdate
	now := time.Now()
	if t.backdateMs > 0 {
		now = now.Add(-time.Duration(t.backdateMs) * time.Millisecond)
	}

	// Extract template timing information from first span to establish trace start
	// Find the span with the earliest start offset (root span)
	var rootSpan *otlptrace.Span
	minOffset := int64(0)
	maxDuration := int64(0)

	for _, span := range spans {
		// Extract template metadata
		startOffset, duration := t.extractSpanTiming(span)

		if rootSpan == nil || startOffset < minOffset {
			minOffset = startOffset
			rootSpan = span
		}

		if duration > maxDuration {
			maxDuration = duration
		}
	}

	// Calculate trace start time (now - total trace duration)
	traceStartNano := now.UnixNano() - maxDuration

	// Apply timestamps to all spans
	for _, span := range spans {
		startOffset, duration := t.extractSpanTiming(span)

		// Calculate absolute start time
		spanStartNano := traceStartNano + startOffset

		// Add jitter if configured
		if t.jitterMs > 0 {
			jitter := rand.Intn(t.jitterMs)
			spanStartNano += int64(jitter) * 1_000_000
		}

		span.StartTimeUnixNano = uint64(spanStartNano)
		span.EndTimeUnixNano = uint64(spanStartNano + duration)

		// Remove template metadata attributes
		span.Attributes = t.removeTemplateAttributes(span.Attributes)
	}
}

// InjectMetricTimestamps adds timestamps to metric data points
func (t *TimestampInjector) InjectMetricTimestamps(metric *otlpmetrics.Metric) {
	now := time.Now()
	if t.backdateMs > 0 {
		now = now.Add(-time.Duration(t.backdateMs) * time.Millisecond)
	}

	// Add jitter
	if t.jitterMs > 0 {
		jitter := rand.Intn(t.jitterMs)
		now = now.Add(time.Duration(jitter) * time.Millisecond)
	}

	nowNano := uint64(now.UnixNano())

	switch data := metric.Data.(type) {
	case *otlpmetrics.Metric_Gauge:
		for _, dp := range data.Gauge.DataPoints {
			dp.TimeUnixNano = nowNano
		}

	case *otlpmetrics.Metric_Sum:
		for _, dp := range data.Sum.DataPoints {
			dp.TimeUnixNano = nowNano
			// For cumulative sums, also set start time
			dp.StartTimeUnixNano = nowNano
		}

	case *otlpmetrics.Metric_Histogram:
		for _, dp := range data.Histogram.DataPoints {
			dp.TimeUnixNano = nowNano
			dp.StartTimeUnixNano = nowNano
		}
	}
}

// InjectLogTimestamps adds timestamps to log records
func (t *TimestampInjector) InjectLogTimestamps(logs []*otlplogs.LogRecord) {
	now := time.Now()
	if t.backdateMs > 0 {
		now = now.Add(-time.Duration(t.backdateMs) * time.Millisecond)
	}

	for _, log := range logs {
		// Add jitter for each log
		logTime := now
		if t.jitterMs > 0 {
			jitter := rand.Intn(t.jitterMs)
			logTime = logTime.Add(time.Duration(jitter) * time.Millisecond)
		}

		nowNano := uint64(logTime.UnixNano())
		log.TimeUnixNano = nowNano
		log.ObservedTimeUnixNano = nowNano
	}
}

// extractSpanTiming extracts timing information from template attributes
func (t *TimestampInjector) extractSpanTiming(span *otlptrace.Span) (startOffset, duration int64) {
	for _, attr := range span.Attributes {
		if attr.Key == "_template.start_offset_nanos" {
			if intVal := attr.Value.GetIntValue(); intVal != 0 {
				startOffset = intVal
			}
		}
		if attr.Key == "_template.duration_nanos" {
			if intVal := attr.Value.GetIntValue(); intVal != 0 {
				duration = intVal
			}
		}
	}

	// Default duration if not found
	if duration == 0 {
		duration = 10_000_000 // 10ms default
	}

	return startOffset, duration
}

// removeTemplateAttributes removes template metadata from attributes
func (t *TimestampInjector) removeTemplateAttributes(attrs []*commonpb.KeyValue) []*commonpb.KeyValue {
	filtered := make([]*commonpb.KeyValue, 0, len(attrs))

	for _, attr := range attrs {
		if attr.Key != "_template.start_offset_nanos" && attr.Key != "_template.duration_nanos" {
			filtered = append(filtered, attr)
		}
	}

	return filtered
}
