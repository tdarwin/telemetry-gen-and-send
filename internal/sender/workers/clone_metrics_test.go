package workers

import (
	"testing"

	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func gaugeMetricTemplate() *otlpcollectormetrics.ExportMetricsServiceRequest {
	dp := &otlpmetrics.NumberDataPoint{
		Attributes: []*commonpb.KeyValue{strAttr("host", "a")},
		Value:      &otlpmetrics.NumberDataPoint_AsInt{AsInt: 1},
	}
	metric := &otlpmetrics.Metric{
		Name: "requests",
		Data: &otlpmetrics.Metric_Gauge{Gauge: &otlpmetrics.Gauge{DataPoints: []*otlpmetrics.NumberDataPoint{dp}}},
	}
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{{
			ScopeMetrics: []*otlpmetrics.ScopeMetrics{{Metrics: []*otlpmetrics.Metric{metric}}},
		}},
	}
}

// TestCloneMetricsRequestIndependentDataPoints proves clones no longer share
// the underlying NumberDataPoint objects with each other or with the
// template. This is a regression guard for a pre-existing bug: the old
// cloneMetricsRequest copied a metric's Data oneof by reference, so every
// concurrent worker goroutine and every sending.multiplier replay mutated the
// exact same *NumberDataPoint objects via InjectMetricTimestamps.
func TestCloneMetricsRequestIndependentDataPoints(t *testing.T) {
	template := gaugeMetricTemplate()

	cloneA := cloneMetricsRequest(template)
	cloneB := cloneMetricsRequest(template)

	dpA := cloneA.ResourceMetrics[0].ScopeMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_Gauge).Gauge.DataPoints[0]
	dpB := cloneB.ResourceMetrics[0].ScopeMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_Gauge).Gauge.DataPoints[0]
	dpTemplate := template.ResourceMetrics[0].ScopeMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_Gauge).Gauge.DataPoints[0]

	if dpA == dpB {
		t.Fatal("cloneA and cloneB share the same *NumberDataPoint object")
	}
	if dpA == dpTemplate || dpB == dpTemplate {
		t.Fatal("a clone shares the same *NumberDataPoint object as the template")
	}

	// Mutate clone A's timestamp and attributes; clone B and the template
	// must be completely unaffected.
	dpA.TimeUnixNano = 12345
	dpA.Attributes = append(dpA.Attributes, strAttr("meta.sequence_id_test", "0"))

	if dpB.TimeUnixNano == 12345 {
		t.Error("mutating cloneA's TimeUnixNano leaked into cloneB")
	}
	if dpTemplate.TimeUnixNano == 12345 {
		t.Error("mutating cloneA's TimeUnixNano leaked into the template")
	}
	if len(dpB.Attributes) != 1 {
		t.Errorf("cloneB.Attributes length = %d, want 1 (must not see cloneA's appended attribute)", len(dpB.Attributes))
	}
	if len(dpTemplate.Attributes) != 1 {
		t.Errorf("template.Attributes length = %d, want 1 (must not see cloneA's appended attribute)", len(dpTemplate.Attributes))
	}
}
