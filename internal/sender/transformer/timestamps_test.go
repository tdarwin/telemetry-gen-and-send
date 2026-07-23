package transformer

import (
	"testing"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func intAttr(key string, v int64) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: v}}}
}

func strAttr(key, v string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}}}
}

func TestExtractEmitDelayMs(t *testing.T) {
	inj := NewTimestampInjector(0, 0)

	withDelay := &otlptrace.Span{Attributes: []*commonpb.KeyValue{
		intAttr("_template.emit_delay_ms", 90000),
		strAttr("service.name", "svc"),
	}}
	if got := inj.ExtractEmitDelayMs(withDelay); got != 90000 {
		t.Errorf("ExtractEmitDelayMs = %d, want 90000", got)
	}

	without := &otlptrace.Span{Attributes: []*commonpb.KeyValue{strAttr("service.name", "svc")}}
	if got := inj.ExtractEmitDelayMs(without); got != 0 {
		t.Errorf("ExtractEmitDelayMs = %d, want 0", got)
	}
}

// TestInjectStripsTemplateAttrs verifies that all _template.* attributes,
// including emit_delay_ms, are removed while real attributes are kept and
// timestamps are set.
func TestInjectStripsTemplateAttrs(t *testing.T) {
	inj := NewTimestampInjector(0, 0)

	span := &otlptrace.Span{Attributes: []*commonpb.KeyValue{
		intAttr("_template.start_offset_nanos", 0),
		intAttr("_template.duration_nanos", 1_000_000),
		intAttr("_template.emit_delay_ms", 90000),
		strAttr("service.name", "svc"),
	}}

	inj.InjectSpanTimestamps([]*otlptrace.Span{span})

	for _, a := range span.Attributes {
		switch a.Key {
		case "_template.start_offset_nanos", "_template.duration_nanos", "_template.emit_delay_ms":
			t.Errorf("template attribute %q was not stripped", a.Key)
		}
	}
	if len(span.Attributes) != 1 || span.Attributes[0].Key != "service.name" {
		t.Errorf("expected only service.name to remain, got %v", span.Attributes)
	}
	if span.StartTimeUnixNano == 0 || span.EndTimeUnixNano == 0 {
		t.Errorf("timestamps not injected: start=%d end=%d", span.StartTimeUnixNano, span.EndTimeUnixNano)
	}
}
