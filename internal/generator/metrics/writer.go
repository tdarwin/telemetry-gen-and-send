package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

// MetricTemplate represents a metric with its time series
type MetricTemplate struct {
	Definition     MetricDefinition
	DimensionSets  []DimensionSet
}

// MetricsWriter handles writing metric templates to disk
type MetricsWriter struct {
	outputDir string
	prefix    string
}

// NewMetricsWriter creates a new metrics writer
func NewMetricsWriter(outputDir, prefix string) *MetricsWriter {
	return &MetricsWriter{
		outputDir: outputDir,
		prefix:    prefix,
	}
}

// WriteMetrics writes metric templates to protobuf and optionally JSON
func (w *MetricsWriter) WriteMetrics(metrics []*MetricTemplate, writeJSON bool) error {
	// Ensure output directory exists
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert metrics to OTLP format
	request := w.metricsToOTLP(metrics)

	// Write protobuf
	pbPath := filepath.Join(w.outputDir, fmt.Sprintf("%s-metrics.pb", w.prefix))
	if err := w.writeProtobuf(request, pbPath); err != nil {
		return fmt.Errorf("failed to write protobuf: %w", err)
	}

	// Count time series
	totalTimeSeries := 0
	for _, m := range metrics {
		totalTimeSeries += len(m.DimensionSets)
	}

	fmt.Printf("Wrote %d metrics (%d time series) to %s\n", len(metrics), totalTimeSeries, pbPath)

	// Write JSON if requested
	if writeJSON {
		jsonPath := filepath.Join(w.outputDir, fmt.Sprintf("%s-metrics.json", w.prefix))
		if err := w.writeJSON(request, jsonPath); err != nil {
			return fmt.Errorf("failed to write JSON: %w", err)
		}
		fmt.Printf("Wrote metrics JSON to %s\n", jsonPath)
	}

	return nil
}

// metricsToOTLP converts metric templates to OTLP ExportMetricsServiceRequest
func (w *MetricsWriter) metricsToOTLP(metrics []*MetricTemplate) *otlpcollectormetrics.ExportMetricsServiceRequest {
	request := &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key: "service.name",
							Value: &commonpb.AnyValue{
								Value: &commonpb.AnyValue_StringValue{
									StringValue: "telemetry-generator",
								},
							},
						},
					},
				},
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope: &commonpb.InstrumentationScope{
							Name:    "telemetry-generator",
							Version: "1.0.0",
						},
						Metrics: make([]*otlpmetrics.Metric, 0),
					},
				},
			},
		},
	}

	// Convert each metric template
	for _, metricTemplate := range metrics {
		otlpMetric := w.templateToOTLP(metricTemplate)
		request.ResourceMetrics[0].ScopeMetrics[0].Metrics = append(
			request.ResourceMetrics[0].ScopeMetrics[0].Metrics,
			otlpMetric,
		)
	}

	return request
}

// templateToOTLP converts a metric template to OTLP Metric
func (w *MetricsWriter) templateToOTLP(template *MetricTemplate) *otlpmetrics.Metric {
	metric := &otlpmetrics.Metric{
		Name:        template.Definition.Name,
		Description: template.Definition.Description,
		Unit:        template.Definition.Unit,
	}

	// Create data points for each dimension set
	switch template.Definition.Type {
	case MetricTypeGauge:
		metric.Data = &otlpmetrics.Metric_Gauge{
			Gauge: &otlpmetrics.Gauge{
				DataPoints: w.createGaugeDataPoints(template),
			},
		}

	case MetricTypeSum:
		metric.Data = &otlpmetrics.Metric_Sum{
			Sum: &otlpmetrics.Sum{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				IsMonotonic:            true,
				DataPoints:             w.createSumDataPoints(template),
			},
		}

	case MetricTypeHistogram:
		metric.Data = &otlpmetrics.Metric_Histogram{
			Histogram: &otlpmetrics.Histogram{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				DataPoints:             w.createHistogramDataPoints(template),
			},
		}
	}

	return metric
}

// createGaugeDataPoints creates gauge data points
func (w *MetricsWriter) createGaugeDataPoints(template *MetricTemplate) []*otlpmetrics.NumberDataPoint {
	dataPoints := make([]*otlpmetrics.NumberDataPoint, 0, len(template.DimensionSets))
	minVal, maxVal := template.Definition.GetValueRange()

	for _, dimSet := range template.DimensionSets {
		value := common.RandomFloat64(minVal, maxVal)

		dp := &otlpmetrics.NumberDataPoint{
			Attributes:   dimSet.ToAttributes(),
			TimeUnixNano: 0, // No timestamp in template
		}
		dp.Value = &otlpmetrics.NumberDataPoint_AsDouble{AsDouble: value}

		dataPoints = append(dataPoints, dp)
	}

	return dataPoints
}

// createSumDataPoints creates sum data points
func (w *MetricsWriter) createSumDataPoints(template *MetricTemplate) []*otlpmetrics.NumberDataPoint {
	dataPoints := make([]*otlpmetrics.NumberDataPoint, 0, len(template.DimensionSets))
	minVal, maxVal := template.Definition.GetValueRange()

	for _, dimSet := range template.DimensionSets {
		// For sums, use integer values
		value := float64(common.RandomInt64(int64(minVal), int64(maxVal)))

		dp := &otlpmetrics.NumberDataPoint{
			Attributes:   dimSet.ToAttributes(),
			TimeUnixNano: 0, // No timestamp in template
		}
		dp.Value = &otlpmetrics.NumberDataPoint_AsInt{AsInt: int64(value)}

		dataPoints = append(dataPoints, dp)
	}

	return dataPoints
}

// createHistogramDataPoints creates histogram data points
func (w *MetricsWriter) createHistogramDataPoints(template *MetricTemplate) []*otlpmetrics.HistogramDataPoint {
	dataPoints := make([]*otlpmetrics.HistogramDataPoint, 0, len(template.DimensionSets))

	for _, dimSet := range template.DimensionSets {
		// Generate histogram buckets
		buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
		counts := make([]uint64, len(buckets)+1)

		// Generate random counts for buckets
		for i := range counts {
			counts[i] = uint64(common.RandomInt(10, 1000))
		}

		sum := 0.0
		count := uint64(0)
		for i, c := range counts {
			count += c
			if i < len(buckets) {
				sum += float64(c) * buckets[i]
			}
		}

		dp := &otlpmetrics.HistogramDataPoint{
			Attributes:       dimSet.ToAttributes(),
			TimeUnixNano:     0, // No timestamp in template
			Count:            count,
			Sum:              &sum,
			BucketCounts:     counts,
			ExplicitBounds:   buckets,
		}

		dataPoints = append(dataPoints, dp)
	}

	return dataPoints
}

// writeProtobuf writes the OTLP request as protobuf binary
func (w *MetricsWriter) writeProtobuf(request *otlpcollectormetrics.ExportMetricsServiceRequest, path string) error {
	data, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// writeJSON writes the OTLP request as JSON
func (w *MetricsWriter) writeJSON(request *otlpcollectormetrics.ExportMetricsServiceRequest, path string) error {
	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
