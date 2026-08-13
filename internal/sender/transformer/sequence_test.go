package transformer

import (
	"sync"
	"testing"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlplogs "go.opentelemetry.io/proto/otlp/logs/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func gaugeMetric(dpCount int) *otlpmetrics.Metric {
	dps := make([]*otlpmetrics.NumberDataPoint, dpCount)
	for i := range dps {
		dps[i] = &otlpmetrics.NumberDataPoint{}
	}
	return &otlpmetrics.Metric{Data: &otlpmetrics.Metric_Gauge{Gauge: &otlpmetrics.Gauge{DataPoints: dps}}}
}

func seqValue(key string, attrs []*commonpb.KeyValue) (int64, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value.GetIntValue(), true
		}
	}
	return 0, false
}

func TestSequenceInjectorDisabledNoop(t *testing.T) {
	inj := NewSequenceInjector(false, "meta.sequence_id")

	spans := []*otlptrace.Span{{}}
	inj.StampSpans(spans)
	if _, ok := seqValue("meta.sequence_id", spans[0].Attributes); ok {
		t.Error("StampSpans added an attribute while disabled")
	}

	metric := gaugeMetric(1)
	inj.StampMetricDataPoints(metric)
	dp := metric.Data.(*otlpmetrics.Metric_Gauge).Gauge.DataPoints[0]
	if _, ok := seqValue("meta.sequence_id", dp.Attributes); ok {
		t.Error("StampMetricDataPoints added an attribute while disabled")
	}

	logs := []*otlplogs.LogRecord{{}}
	inj.StampLogRecords(logs)
	if _, ok := seqValue("meta.sequence_id", logs[0].Attributes); ok {
		t.Error("StampLogRecords added an attribute while disabled")
	}
}

func TestSequenceInjectorPerSignalIndependentCounters(t *testing.T) {
	inj := NewSequenceInjector(true, "meta.sequence_id")

	spans := []*otlptrace.Span{{}, {}, {}}
	inj.StampSpans(spans)
	for i, sp := range spans {
		v, ok := seqValue("meta.sequence_id", sp.Attributes)
		if !ok || v != int64(i) {
			t.Errorf("span %d sequence = %v (ok=%v), want %d", i, v, ok, i)
		}
	}

	metric := gaugeMetric(3)
	inj.StampMetricDataPoints(metric)
	for i, dp := range metric.Data.(*otlpmetrics.Metric_Gauge).Gauge.DataPoints {
		v, ok := seqValue("meta.sequence_id", dp.Attributes)
		if !ok || v != int64(i) {
			t.Errorf("data point %d sequence = %v (ok=%v), want %d (metrics counter must start at 0 independent of traces)", i, v, ok, i)
		}
	}

	logs := []*otlplogs.LogRecord{{}, {}}
	inj.StampLogRecords(logs)
	for i, lr := range logs {
		v, ok := seqValue("meta.sequence_id", lr.Attributes)
		if !ok || v != int64(i) {
			t.Errorf("log %d sequence = %v (ok=%v), want %d (logs counter must start at 0 independent of traces/metrics)", i, v, ok, i)
		}
	}
}

// TestSequenceInjectorMonotonicAcrossReplays simulates what sending.multiplier
// replay does: the same injector is reused across several independent
// "replay" batches of freshly rebuilt spans. Sequence numbers must stay
// contiguous and never repeat across replays — the entire point of assigning
// them at send time rather than baking them into the on-disk template.
func TestSequenceInjectorMonotonicAcrossReplays(t *testing.T) {
	inj := NewSequenceInjector(true, "meta.sequence_id")

	var allSeen []int64
	for replay := 0; replay < 5; replay++ {
		spans := []*otlptrace.Span{{}, {}} // fresh spans each "replay", like a re-cloned template
		inj.StampSpans(spans)
		for _, sp := range spans {
			v, ok := seqValue("meta.sequence_id", sp.Attributes)
			if !ok {
				t.Fatalf("replay %d: span missing sequence attribute", replay)
			}
			allSeen = append(allSeen, v)
		}
	}

	for i, v := range allSeen {
		if v != int64(i) {
			t.Errorf("sequence[%d] = %d, want %d (must be contiguous across replays, no resets)", i, v, i)
		}
	}
}

// TestSequenceInjectorConcurrentUnique proves the counters are safe under
// concurrent use from many worker goroutines (run with -race).
func TestSequenceInjectorConcurrentUnique(t *testing.T) {
	const goroutines = 50
	const perGoroutine = 200

	inj := NewSequenceInjector(true, "meta.sequence_id")

	var mu sync.Mutex
	seen := make(map[int64]bool, goroutines*perGoroutine)

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				spans := []*otlptrace.Span{{}}
				inj.StampSpans(spans)
				v, ok := seqValue("meta.sequence_id", spans[0].Attributes)
				if !ok {
					mu.Lock()
					t.Error("missing sequence attribute")
					mu.Unlock()
					return
				}
				mu.Lock()
				if seen[v] {
					t.Errorf("duplicate sequence value %d", v)
				}
				seen[v] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(seen) != goroutines*perGoroutine {
		t.Errorf("saw %d unique values, want %d", len(seen), goroutines*perGoroutine)
	}
	for i := 0; i < goroutines*perGoroutine; i++ {
		if !seen[int64(i)] {
			t.Errorf("sequence value %d never assigned (gap in the range)", i)
		}
	}
}

// TestAppendSeqAttrDoesNotAliasSharedBackingArray reproduces the
// shared-template-backing-array bug class directly: two "clones" that both
// start from the same underlying array (as the trace/logs clone helpers
// alias Attributes from a template) must not corrupt each other or the
// original when stamped.
func TestAppendSeqAttrDoesNotAliasSharedBackingArray(t *testing.T) {
	orig := make([]*commonpb.KeyValue, 1, 4) // spare capacity, like a template slice
	orig[0] = &commonpb.KeyValue{Key: "service.name"}

	inj := NewSequenceInjector(true, "meta.sequence_id")

	spanA := &otlptrace.Span{Attributes: orig} // "clone" 1, aliases orig's backing array
	spanB := &otlptrace.Span{Attributes: orig} // "clone" 2, same alias

	inj.StampSpans([]*otlptrace.Span{spanA})
	inj.StampSpans([]*otlptrace.Span{spanB})

	if len(spanA.Attributes) != 2 || len(spanB.Attributes) != 2 {
		t.Fatalf("expected 2 attributes on each stamped span, got %d and %d", len(spanA.Attributes), len(spanB.Attributes))
	}
	if spanA.Attributes[1] == spanB.Attributes[1] {
		t.Error("both spans' appended attribute is the same object — they landed in the same backing array slot")
	}
	vA, _ := seqValue("meta.sequence_id", spanA.Attributes)
	vB, _ := seqValue("meta.sequence_id", spanB.Attributes)
	if vA == vB {
		t.Error("both spans ended up with the same sequence value — one overwrote the other's slot")
	}

	// The original backing array beyond its original length must be untouched.
	probe := orig[:2]
	if probe[1] != nil {
		t.Error("appendSeqAttr wrote into orig's own backing array beyond its original length — this is the exact corruption bug being guarded against")
	}
}
