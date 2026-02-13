package traces

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

// SpanNode represents a span in a trace tree
type SpanNode struct {
	SpanID     []byte
	ParentID   []byte
	Service    *ServiceNode
	Operation  Operation
	Duration   int64 // in nanoseconds
	StartTime  int64 // relative offset from trace start (we'll add timestamps later)
	Attributes []*commonpb.KeyValue
	Children   []*SpanNode
}

// TraceTemplate represents a complete trace without timestamps
type TraceTemplate struct {
	TraceID   []byte
	RootSpan  *SpanNode
	SpanCount int
}

// SpanGenerator generates spans for traces
type SpanGenerator struct {
	config   *config.TracesConfig
	topology *ServiceTopology
	customAttrs []common.AttributeSchema
}

// NewSpanGenerator creates a new span generator
func NewSpanGenerator(cfg *config.TracesConfig, topology *ServiceTopology) *SpanGenerator {
	return &SpanGenerator{
		config:      cfg,
		topology:    topology,
		customAttrs: common.GenerateCustomAttributeSchemas(cfg.CustomAttributes.Count),
	}
}

// GenerateTrace generates a complete trace
func (g *SpanGenerator) GenerateTrace() *TraceTemplate {
	// Determine span count using normal distribution
	spanCount := common.NormalInt(g.config.Spans.AvgPerTrace, g.config.Spans.StdDev)

	trace := &TraceTemplate{
		TraceID:   generateTraceID(),
		SpanCount: spanCount,
	}

	// Start with ingress service
	ingressService := g.topology.GetRandomIngress()
	if ingressService == nil {
		// Fallback to first service
		ingressService = g.topology.Services[0]
	}

	// Create root span
	rootOp := ingressService.GetRandomOperation()
	trace.RootSpan = &SpanNode{
		SpanID:     generateSpanID(),
		ParentID:   nil,
		Service:    ingressService,
		Operation:  rootOp,
		Duration:   0, // Will be calculated after building tree
		StartTime:  0, // Root always starts at 0
		Attributes: g.generateAttributes(ingressService, rootOp),
		Children:   make([]*SpanNode, 0),
	}

	// Build the rest of the tree
	remainingSpans := spanCount - 1
	g.buildSpanTree(trace.RootSpan, remainingSpans, 0)

	// Calculate durations bottom-up
	g.calculateDurations(trace.RootSpan)

	return trace
}

// GenerateHighSpanTrace generates a trace with a very high span count
func (g *SpanGenerator) GenerateHighSpanTrace(spanCount int) *TraceTemplate {
	trace := &TraceTemplate{
		TraceID:   generateTraceID(),
		SpanCount: spanCount,
	}

	ingressService := g.topology.GetRandomIngress()
	if ingressService == nil {
		ingressService = g.topology.Services[0]
	}

	rootOp := ingressService.GetRandomOperation()
	trace.RootSpan = &SpanNode{
		SpanID:     generateSpanID(),
		ParentID:   nil,
		Service:    ingressService,
		Operation:  rootOp,
		Duration:   0,
		StartTime:  0,
		Attributes: g.generateAttributes(ingressService, rootOp),
		Children:   make([]*SpanNode, 0),
	}

	// For high span count, use a wider tree structure
	remainingSpans := spanCount - 1
	g.buildWideSpanTree(trace.RootSpan, remainingSpans, 0)

	g.calculateDurations(trace.RootSpan)

	return trace
}

// buildSpanTree recursively builds a span tree
func (g *SpanGenerator) buildSpanTree(parent *SpanNode, remainingSpans int, depth int) int {
	if remainingSpans <= 0 || depth > 10 { // Limit depth to prevent stack overflow
		return 0
	}

	// Determine how many children this span should have
	childCount := common.RandomInt(1, 4)
	if childCount > remainingSpans {
		childCount = remainingSpans
	}

	spansCreated := 0
	currentOffset := parent.StartTime

	for i := 0; i < childCount && spansCreated < remainingSpans; i++ {
		// Determine service for child span
		var childService *ServiceNode
		var childOp Operation

		// 70% chance to call downstream service, 30% chance to call within same service
		if parent.Service.HasDownstream() && common.RandomInt(1, 100) <= 70 {
			childService = parent.Service.GetRandomDownstream()
		} else {
			childService = parent.Service
		}

		childOp = childService.GetRandomOperation()

		// Create child span
		child := &SpanNode{
			SpanID:     generateSpanID(),
			ParentID:   parent.SpanID,
			Service:    childService,
			Operation:  childOp,
			Duration:   0,
			StartTime:  currentOffset,
			Attributes: g.generateAttributes(childService, childOp),
			Children:   make([]*SpanNode, 0),
		}

		parent.Children = append(parent.Children, child)
		spansCreated++

		// Recursively build children for this child
		childBudget := (remainingSpans - spansCreated) / (childCount - i)
		if childBudget < 1 {
			childBudget = remainingSpans - spansCreated
		}

		childSpans := g.buildSpanTree(child, childBudget, depth+1)
		spansCreated += childSpans

		// Offset for next sibling (sequential execution)
		currentOffset += child.Duration
	}

	return spansCreated
}

// buildWideSpanTree builds a wider tree for high span count traces
func (g *SpanGenerator) buildWideSpanTree(parent *SpanNode, remainingSpans int, depth int) int {
	if remainingSpans <= 0 || depth > 20 {
		return 0
	}

	// For high span counts, create more children per level
	childCount := common.RandomInt(5, 15)
	if childCount > remainingSpans {
		childCount = remainingSpans
	}

	spansCreated := 0

	for i := 0; i < childCount && spansCreated < remainingSpans; i++ {
		var childService *ServiceNode
		if parent.Service.HasDownstream() && common.RandomBool() {
			childService = parent.Service.GetRandomDownstream()
		} else {
			childService = parent.Service
		}

		childOp := childService.GetRandomOperation()

		child := &SpanNode{
			SpanID:     generateSpanID(),
			ParentID:   parent.SpanID,
			Service:    childService,
			Operation:  childOp,
			Duration:   common.RandomDuration(1000000, 50000000), // 1-50ms
			StartTime:  parent.StartTime,
			Attributes: g.generateAttributes(childService, childOp),
			Children:   make([]*SpanNode, 0),
		}

		parent.Children = append(parent.Children, child)
		spansCreated++

		// Recursively build with remaining budget
		if spansCreated < remainingSpans {
			childBudget := (remainingSpans - spansCreated) / (childCount - i)
			childSpans := g.buildWideSpanTree(child, childBudget, depth+1)
			spansCreated += childSpans
		}
	}

	return spansCreated
}

// calculateDurations calculates durations for all spans bottom-up
func (g *SpanGenerator) calculateDurations(span *SpanNode) int64 {
	if len(span.Children) == 0 {
		// Leaf span - generate random duration
		span.Duration = common.RandomDuration(1000000, 100000000) // 1-100ms in nanoseconds
		return span.Duration
	}

	// Calculate children durations first
	totalChildDuration := int64(0)
	for _, child := range span.Children {
		childDuration := g.calculateDurations(child)
		totalChildDuration += childDuration
	}

	// Parent duration is children duration plus some overhead
	overhead := common.RandomDuration(500000, 5000000) // 0.5-5ms overhead
	span.Duration = totalChildDuration + overhead

	return span.Duration
}

// generateAttributes generates attributes for a span
func (g *SpanGenerator) generateAttributes(service *ServiceNode, op Operation) []*commonpb.KeyValue {
	attrs := make([]*commonpb.KeyValue, 0)

	// Add service name
	attrs = append(attrs, common.CreateStringAttribute("service.name", service.Name))

	// Add operation-specific attributes
	switch op.Type {
	case OperationTypeHTTP:
		httpAttrs := common.CreateHTTPAttributes(op.HTTPMethod, op.HTTPPath, common.RandomHTTPStatus())
		attrs = append(attrs, httpAttrs...)

	case OperationTypeDB:
		dbAttrs := common.CreateDBAttributes(op.DBSystem, op.DBStatement)
		attrs = append(attrs, dbAttrs...)

	case OperationTypeInternal:
		attrs = append(attrs, common.CreateStringAttribute("span.kind", "internal"))
	}

	// Randomly add custom attributes (30% chance)
	if common.RandomInt(1, 100) <= 30 && len(g.customAttrs) > 0 {
		numCustom := common.RandomInt(1, 3)
		for i := 0; i < numCustom && i < len(g.customAttrs); i++ {
			schema := common.RandomChoice(g.customAttrs)
			attrs = append(attrs, common.CreateAttribute(schema))
		}
	}

	return attrs
}

// generateTraceID generates a random trace ID (16 bytes)
func generateTraceID() []byte {
	id := make([]byte, 16)
	rand.Read(id)
	return id
}

// generateSpanID generates a random span ID (8 bytes)
func generateSpanID() []byte {
	id := make([]byte, 8)
	rand.Read(id)
	return id
}

// ToOTLPSpan converts a SpanNode to an OTLP Span
func (s *SpanNode) ToOTLPSpan() *otlptrace.Span {
	span := &otlptrace.Span{
		TraceId:           nil, // Will be set by caller
		SpanId:            s.SpanID,
		ParentSpanId:      s.ParentID,
		Name:              s.Operation.Name,
		Kind:              otlptrace.Span_SPAN_KIND_INTERNAL,
		StartTimeUnixNano: 0, // No timestamp in template
		EndTimeUnixNano:   0, // No timestamp in template
		Attributes:        s.Attributes,
		Status: &otlptrace.Status{
			Code: otlptrace.Status_STATUS_CODE_OK,
		},
	}

	// Set span kind based on operation type
	switch s.Operation.Type {
	case OperationTypeHTTP:
		if s.ParentID == nil {
			span.Kind = otlptrace.Span_SPAN_KIND_SERVER
		} else {
			span.Kind = otlptrace.Span_SPAN_KIND_CLIENT
		}
	case OperationTypeDB:
		span.Kind = otlptrace.Span_SPAN_KIND_CLIENT
	}

	return span
}

// CollectSpans collects all spans from the tree into a flat list
func (t *TraceTemplate) CollectSpans() []*SpanNode {
	spans := make([]*SpanNode, 0, t.SpanCount)
	t.collectSpansRecursive(t.RootSpan, &spans)
	return spans
}

func (t *TraceTemplate) collectSpansRecursive(span *SpanNode, spans *[]*SpanNode) {
	*spans = append(*spans, span)
	for _, child := range span.Children {
		t.collectSpansRecursive(child, spans)
	}
}

// TraceIDToString converts a trace ID to a hex string
func TraceIDToString(traceID []byte) string {
	return hex.EncodeToString(traceID)
}

// SpanIDToString converts a span ID to a hex string
func SpanIDToString(spanID []byte) string {
	return hex.EncodeToString(spanID)
}
