package transformer

import (
	"sync/atomic"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlplogs "go.opentelemetry.io/proto/otlp/logs/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

// SequenceInjector stamps a monotonically increasing sequence-number
// attribute onto every span, metric data point, and log record actually
// sent, when enabled. Each signal type has its own independent counter
// starting at 0, incremented atomically so it stays correct across
// concurrent worker goroutines and every sending.multiplier replay. When
// disabled, all Stamp* methods are no-ops, so callers can invoke them
// unconditionally.
type SequenceInjector struct {
	enabled bool
	key     string
	traces  atomic.Uint64
	metrics atomic.Uint64
	logs    atomic.Uint64
}

// NewSequenceInjector creates a SequenceInjector. attributeKey is the
// attribute key to stamp (e.g. "meta.sequence_id"); ignored when enabled is
// false.
func NewSequenceInjector(enabled bool, attributeKey string) *SequenceInjector {
	return &SequenceInjector{enabled: enabled, key: attributeKey}
}

// StampSpans appends the sequence attribute to every span's Attributes. Must
// be called once per span actually scheduled for export — including spans
// later routed to the deferred scheduler — before a trace's spans are
// partitioned into immediate/deferred sets.
func (s *SequenceInjector) StampSpans(spans []*otlptrace.Span) {
	if !s.enabled {
		return
	}
	for _, sp := range spans {
		sp.Attributes = appendSeqAttr(sp.Attributes, s.key, s.traces.Add(1)-1)
	}
}

// StampMetricDataPoints appends the sequence attribute to every data point of
// the given metric (Gauge, Sum, or Histogram).
func (s *SequenceInjector) StampMetricDataPoints(metric *otlpmetrics.Metric) {
	if !s.enabled {
		return
	}
	switch data := metric.Data.(type) {
	case *otlpmetrics.Metric_Gauge:
		for _, dp := range data.Gauge.DataPoints {
			dp.Attributes = appendSeqAttr(dp.Attributes, s.key, s.metrics.Add(1)-1)
		}
	case *otlpmetrics.Metric_Sum:
		for _, dp := range data.Sum.DataPoints {
			dp.Attributes = appendSeqAttr(dp.Attributes, s.key, s.metrics.Add(1)-1)
		}
	case *otlpmetrics.Metric_Histogram:
		for _, dp := range data.Histogram.DataPoints {
			dp.Attributes = appendSeqAttr(dp.Attributes, s.key, s.metrics.Add(1)-1)
		}
	}
}

// StampLogRecords appends the sequence attribute to every log record.
func (s *SequenceInjector) StampLogRecords(records []*otlplogs.LogRecord) {
	if !s.enabled {
		return
	}
	for _, lr := range records {
		lr.Attributes = appendSeqAttr(lr.Attributes, s.key, s.logs.Add(1)-1)
	}
}

// appendSeqAttr returns a NEW slice: orig's elements plus one sequence
// key/value pair. It never appends onto orig's backing array in place —
// orig may be a slice header cloned from an on-disk template and shared,
// via that shared backing array, across every concurrent worker goroutine
// and multiplier replay of the same template. An in-place append with spare
// template capacity would silently corrupt other goroutines'/replays' view
// of that data.
func appendSeqAttr(orig []*commonpb.KeyValue, key string, n uint64) []*commonpb.KeyValue {
	fresh := make([]*commonpb.KeyValue, len(orig), len(orig)+1)
	copy(fresh, orig)
	return append(fresh, &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(n)}},
	})
}
