package workers

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/ratelimit"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/stats"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/transformer"
	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func intAttr(key string, v int64) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: v}}}
}

func strAttr(key, v string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}}}
}

// tmplSpan builds a template span like the generator writes: relative timing in
// _template.* attributes and no absolute timestamps.
func tmplSpan(spanID, parentID []byte, offset, dur, emitDelayMs int64) *otlptrace.Span {
	attrs := []*commonpb.KeyValue{
		intAttr("_template.start_offset_nanos", offset),
		intAttr("_template.duration_nanos", dur),
		strAttr("service.name", "svc"),
	}
	if emitDelayMs > 0 {
		attrs = append(attrs, intAttr("_template.emit_delay_ms", emitDelayMs))
	}
	return &otlptrace.Span{SpanId: spanID, ParentSpanId: parentID, Name: "op", Attributes: attrs}
}

func newTestPool() *WorkerPool {
	return &WorkerPool{
		timestampInjector: transformer.NewTimestampInjector(0, 0),
		idRegenerator:     transformer.NewIDRegenerator(),
	}
}

func oneTraceRS(spans ...*otlptrace.Span) *otlptrace.ResourceSpans {
	for _, s := range spans {
		s.TraceId = []byte("template-traceid")
	}
	return &otlptrace.ResourceSpans{
		ScopeSpans: []*otlptrace.ScopeSpans{{Spans: spans}},
	}
}

func hasTemplateAttr(s *otlptrace.Span) bool {
	for _, a := range s.Attributes {
		switch a.Key {
		case "_template.start_offset_nanos", "_template.duration_nanos", "_template.emit_delay_ms":
			return true
		}
	}
	return false
}

// TestTransformTraceLateRoot verifies that a late root is split off as a
// deferred payload while its children go out immediately, that every emitted
// span shares one regenerated trace ID, and that template attributes are
// stripped.
func TestTransformTraceLateRoot(t *testing.T) {
	p := newTestPool()

	root := tmplSpan([]byte("root0001"), nil, 0, 1_000_000, 90_000) // late root
	c1 := tmplSpan([]byte("child001"), []byte("root0001"), 100, 500_000, 0)
	c2 := tmplSpan([]byte("child002"), []byte("root0001"), 200, 500_000, 0)
	rs := oneTraceRS(root, c1, c2)

	immediate, deferred, immCount := p.transformTrace(rs)

	if immCount != 2 {
		t.Fatalf("immediate span count = %d, want 2", immCount)
	}
	if len(deferred) != 1 {
		t.Fatalf("deferred payloads = %d, want 1", len(deferred))
	}
	if deferred[0].delayMs != 90_000 {
		t.Errorf("deferred delayMs = %d, want 90000", deferred[0].delayMs)
	}
	if deferred[0].spanCount != 1 {
		t.Errorf("deferred spanCount = %d, want 1", deferred[0].spanCount)
	}

	// Collect all emitted spans (immediate + deferred) and check a single trace ID.
	var all []*otlptrace.Span
	for _, ss := range immediate.ScopeSpans {
		all = append(all, ss.Spans...)
	}
	for _, ss := range deferred[0].req.ResourceSpans[0].ScopeSpans {
		all = append(all, ss.Spans...)
	}
	if len(all) != 3 {
		t.Fatalf("total emitted spans = %d, want 3", len(all))
	}
	traceID := all[0].TraceId
	if bytes.Equal(traceID, []byte("template-traceid")) {
		t.Errorf("trace ID was not regenerated")
	}
	for _, s := range all {
		if !bytes.Equal(s.TraceId, traceID) {
			t.Errorf("span %x has trace ID %x, want %x (all spans must share one ID)", s.SpanId, s.TraceId, traceID)
		}
		if hasTemplateAttr(s) {
			t.Errorf("span %x still carries a _template.* attribute after transform", s.SpanId)
		}
	}
}

// TestTransformTracePhantomParentPreserved verifies that a rootless trace's
// phantom parent survives ID regeneration (it is not remapped, since no emitted
// span carries that ID).
func TestTransformTracePhantomParentPreserved(t *testing.T) {
	p := newTestPool()

	phantom := []byte("phantom0")
	root := tmplSpan([]byte("root0001"), phantom, 0, 1_000_000, 0)
	c1 := tmplSpan([]byte("child001"), []byte("root0001"), 100, 500_000, 0)
	rs := oneTraceRS(root, c1)

	immediate, deferred, immCount := p.transformTrace(rs)
	if len(deferred) != 0 {
		t.Fatalf("deferred payloads = %d, want 0", len(deferred))
	}
	if immCount != 2 {
		t.Fatalf("immediate span count = %d, want 2", immCount)
	}

	// Find the root (the span whose parent is not any emitted span's ID).
	emitted := map[string]bool{}
	var spans []*otlptrace.Span
	for _, ss := range immediate.ScopeSpans {
		spans = append(spans, ss.Spans...)
	}
	for _, s := range spans {
		emitted[string(s.SpanId)] = true
	}
	var rootOut *otlptrace.Span
	for _, s := range spans {
		if len(s.ParentSpanId) > 0 && !emitted[string(s.ParentSpanId)] {
			rootOut = s
		}
	}
	if rootOut == nil {
		t.Fatal("no rootless span found after transform")
	}
	if !bytes.Equal(rootOut.ParentSpanId, phantom) {
		t.Errorf("phantom parent = %x, want %x (must be preserved verbatim)", rootOut.ParentSpanId, phantom)
	}
}

// fakeSink records exported requests and the wall-clock time of each export.
type fakeSink struct {
	mu    sync.Mutex
	times []time.Time
	spans []int
}

func (f *fakeSink) Export(_ context.Context, req *otlpcollectortrace.ExportTraceServiceRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.times = append(f.times, time.Now())
	n := 0
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			n += len(ss.Spans)
		}
	}
	f.spans = append(f.spans, n)
	return nil
}

func (f *fakeSink) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.times)
}

func reqWithSpans(n int) *otlpcollectortrace.ExportTraceServiceRequest {
	spans := make([]*otlptrace.Span, n)
	for i := range spans {
		spans[i] = &otlptrace.Span{}
	}
	return &otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{{ScopeSpans: []*otlptrace.ScopeSpans{{Spans: spans}}}},
	}
}

// TestDeferredSchedulerFiresInOrder verifies items fire at (or after) their
// scheduled times, ordered by sendAt regardless of enqueue order, and that
// Close drains everything.
func TestDeferredSchedulerFiresInOrder(t *testing.T) {
	sink := &fakeSink{}
	s := newDeferredScheduler(sink, ratelimit.NewLimiter(0), stats.NewReporter(), 100, 5*time.Second)
	s.Start()

	start := time.Now()
	// Enqueue the later item first to prove ordering is by sendAt, not arrival.
	s.Enqueue(reqWithSpans(1), start.Add(120*time.Millisecond), 1)
	s.Enqueue(reqWithSpans(1), start.Add(40*time.Millisecond), 1)

	dropped := s.Close()
	if dropped != 0 {
		t.Errorf("dropped = %d, want 0", dropped)
	}
	if sink.count() != 2 {
		t.Fatalf("exported %d payloads, want 2", sink.count())
	}
	sink.mu.Lock()
	first, second := sink.times[0], sink.times[1]
	sink.mu.Unlock()
	if !first.After(start.Add(30 * time.Millisecond)) {
		t.Errorf("first export fired too early: %v after start", first.Sub(start))
	}
	if second.Before(first) {
		t.Errorf("exports out of order")
	}
	if !second.After(start.Add(110 * time.Millisecond)) {
		t.Errorf("second export fired too early: %v after start", second.Sub(start))
	}
}

// TestDeferredSchedulerMaxPending verifies overflow enqueues are rejected and
// counted, without starting the consumer loop.
func TestDeferredSchedulerMaxPending(t *testing.T) {
	s := newDeferredScheduler(&fakeSink{}, ratelimit.NewLimiter(0), stats.NewReporter(), 2, time.Second)
	future := time.Now().Add(time.Hour)

	if !s.Enqueue(reqWithSpans(1), future, 1) {
		t.Fatal("first enqueue should succeed")
	}
	if !s.Enqueue(reqWithSpans(1), future, 1) {
		t.Fatal("second enqueue should succeed")
	}
	if s.Enqueue(reqWithSpans(1), future, 3) {
		t.Fatal("third enqueue should be rejected (queue full)")
	}
	if got := s.dropped.Load(); got != 3 {
		t.Errorf("dropped spans = %d, want 3", got)
	}
}
