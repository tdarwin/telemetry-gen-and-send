package transformer

import (
	"crypto/rand"

	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

// IDRegenerator regenerates trace and span IDs while preserving relationships
type IDRegenerator struct{}

// NewIDRegenerator creates a new ID regenerator
func NewIDRegenerator() *IDRegenerator {
	return &IDRegenerator{}
}

// RegenerateTraceIDs regenerates IDs for all spans in a trace
// This preserves parent-child relationships while ensuring uniqueness
func (r *IDRegenerator) RegenerateTraceIDs(spans []*otlptrace.Span) {
	if len(spans) == 0 {
		return
	}

	// Generate new trace ID for the entire trace
	newTraceID := generateTraceID()

	// Map old span IDs to new span IDs
	idMap := make(map[string][]byte)

	// First pass: generate new span IDs for all spans
	for _, span := range spans {
		oldSpanID := string(span.SpanId)
		newSpanID := generateSpanID()
		idMap[oldSpanID] = newSpanID
	}

	// Second pass: update trace IDs, span IDs, and parent span IDs
	for _, span := range spans {
		// Update trace ID
		span.TraceId = newTraceID

		// Update span ID
		oldSpanID := string(span.SpanId)
		span.SpanId = idMap[oldSpanID]

		// Update parent span ID if it exists
		if len(span.ParentSpanId) > 0 {
			oldParentID := string(span.ParentSpanId)
			if newParentID, ok := idMap[oldParentID]; ok {
				span.ParentSpanId = newParentID
			}
		}
	}
}

// generateTraceID generates a random 16-byte trace ID
func generateTraceID() []byte {
	id := make([]byte, 16)
	rand.Read(id)
	return id
}

// generateSpanID generates a random 8-byte span ID
func generateSpanID() []byte {
	id := make([]byte, 8)
	rand.Read(id)
	return id
}
