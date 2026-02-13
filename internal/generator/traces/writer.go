package traces

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

// TraceWriter handles writing trace templates to disk
type TraceWriter struct {
	outputDir string
	prefix    string
}

// NewTraceWriter creates a new trace writer
func NewTraceWriter(outputDir, prefix string) *TraceWriter {
	return &TraceWriter{
		outputDir: outputDir,
		prefix:    prefix,
	}
}

// WriteTraces writes trace templates to protobuf and optionally JSON
func (w *TraceWriter) WriteTraces(traces []*TraceTemplate, writeJSON bool) error {
	// Ensure output directory exists
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert traces to OTLP format
	request := w.tracesToOTLP(traces)

	// Write protobuf
	pbPath := filepath.Join(w.outputDir, fmt.Sprintf("%s-traces.pb", w.prefix))
	if err := w.writeProtobuf(request, pbPath); err != nil {
		return fmt.Errorf("failed to write protobuf: %w", err)
	}

	fmt.Printf("Wrote %d traces to %s\n", len(traces), pbPath)

	// Write JSON if requested
	if writeJSON {
		jsonPath := filepath.Join(w.outputDir, fmt.Sprintf("%s-traces.json", w.prefix))
		if err := w.writeJSON(request, jsonPath); err != nil {
			return fmt.Errorf("failed to write JSON: %w", err)
		}
		fmt.Printf("Wrote trace JSON to %s\n", jsonPath)
	}

	return nil
}

// tracesToOTLP converts trace templates to OTLP ExportTraceServiceRequest
func (w *TraceWriter) tracesToOTLP(traces []*TraceTemplate) *otlpcollectortrace.ExportTraceServiceRequest {
	request := &otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: make([]*otlptrace.ResourceSpans, 0),
	}

	// Each trace becomes its own ResourceSpans to keep all spans together
	// This ensures that cross-service traces remain connected
	for _, trace := range traces {
		spans := trace.CollectSpans()

		// Create ResourceSpans for this trace with a generic resource
		rs := &otlptrace.ResourceSpans{
			Resource: &resourcepb.Resource{
				Attributes: []*commonpb.KeyValue{
					{
						Key: "telemetry.sdk.name",
						Value: &commonpb.AnyValue{
							Value: &commonpb.AnyValue_StringValue{
								StringValue: "telemetry-generator",
							},
						},
					},
					{
						Key: "telemetry.sdk.version",
						Value: &commonpb.AnyValue{
							Value: &commonpb.AnyValue_StringValue{
								StringValue: "1.0.0",
							},
						},
					},
				},
			},
			ScopeSpans: []*otlptrace.ScopeSpans{
				{
					Scope: &commonpb.InstrumentationScope{
						Name:    "telemetry-generator",
						Version: "1.0.0",
					},
					Spans: make([]*otlptrace.Span, 0, len(spans)),
				},
			},
		}

		// Add all spans from this trace
		for _, spanNode := range spans {
			// Convert span to OTLP
			otlpSpan := spanNode.ToOTLPSpan()
			otlpSpan.TraceId = trace.TraceID

			// Service name is already in the span attributes (added by generateAttributes)
			// No need to add it to resource

			// Store duration in attributes since we can't use timestamps
			// This allows the sender to reconstruct relative timings
			otlpSpan.Attributes = append(otlpSpan.Attributes, &commonpb.KeyValue{
				Key: "_template.duration_nanos",
				Value: &commonpb.AnyValue{
					Value: &commonpb.AnyValue_IntValue{
						IntValue: spanNode.Duration,
					},
				},
			})

			// Store start offset for relative timing
			otlpSpan.Attributes = append(otlpSpan.Attributes, &commonpb.KeyValue{
				Key: "_template.start_offset_nanos",
				Value: &commonpb.AnyValue{
					Value: &commonpb.AnyValue_IntValue{
						IntValue: spanNode.StartTime,
					},
				},
			})

			rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, otlpSpan)
		}

		request.ResourceSpans = append(request.ResourceSpans, rs)
	}

	return request
}

// writeProtobuf writes the OTLP request as protobuf binary
func (w *TraceWriter) writeProtobuf(request *otlpcollectortrace.ExportTraceServiceRequest, path string) error {
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
func (w *TraceWriter) writeJSON(request *otlpcollectortrace.ExportTraceServiceRequest, path string) error {
	// Convert to JSON-friendly format
	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetStats returns statistics about the generated traces
type TraceStats struct {
	TotalTraces int
	TotalSpans  int
	AvgSpans    float64
	MinSpans    int
	MaxSpans    int
}

// CalculateStats calculates statistics from a set of traces
func CalculateStats(traces []*TraceTemplate) TraceStats {
	stats := TraceStats{
		TotalTraces: len(traces),
		MinSpans:    int(^uint(0) >> 1), // Max int
		MaxSpans:    0,
	}

	for _, trace := range traces {
		stats.TotalSpans += trace.SpanCount

		if trace.SpanCount < stats.MinSpans {
			stats.MinSpans = trace.SpanCount
		}
		if trace.SpanCount > stats.MaxSpans {
			stats.MaxSpans = trace.SpanCount
		}
	}

	if stats.TotalTraces > 0 {
		stats.AvgSpans = float64(stats.TotalSpans) / float64(stats.TotalTraces)
	}

	return stats
}
