package traces

import (
	"testing"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func resourceAttr(rs *otlptrace.ResourceSpans, key string) (string, bool) {
	for _, a := range rs.Resource.Attributes {
		if a.Key == key {
			return a.Value.GetStringValue(), true
		}
	}
	return "", false
}

// TestResourceHasEntryPointServiceName verifies each trace's ResourceSpans
// carries a resource-level service.name equal to the trace's entry-point (root)
// service — without which backends route everything to unknown_service.
func TestResourceHasEntryPointServiceName(t *testing.T) {
	cfg := &config.TracesConfig{
		Count:    3,
		Spans:    config.SpansConfig{AvgPerTrace: 6, StdDev: 0},
		Services: config.ServicesConfig{Count: 3, Names: []string{"api", "worker", "db"}},
	}
	g := NewSpanGenerator(cfg, testTopology())
	w := NewTraceWriter("/tmp", "test")

	for i := 0; i < 20; i++ {
		tr := g.GenerateTrace()
		req := w.tracesToOTLP([]*TraceTemplate{tr})
		if len(req.ResourceSpans) != 1 {
			t.Fatalf("expected 1 ResourceSpans per trace, got %d", len(req.ResourceSpans))
		}
		got, ok := resourceAttr(req.ResourceSpans[0], "service.name")
		if !ok || got == "" {
			t.Fatal("resource is missing service.name (would route to unknown_service)")
		}
		if got != tr.RootSpan.Service.Name {
			t.Fatalf("resource service.name = %q, want entry-point service %q", got, tr.RootSpan.Service.Name)
		}
	}
}

// TestResourceHasNamespaceWhenSet verifies service.namespace is promoted to the
// resource when the entry-point service has one.
func TestResourceHasNamespaceWhenSet(t *testing.T) {
	cfg := &config.TracesConfig{
		Count: 1,
		Spans: config.SpansConfig{AvgPerTrace: 4, StdDev: 0},
		Services: config.ServicesConfig{
			Count:      2,
			Names:      []string{"api", "db"},
			Namespaces: []string{"shop"},
		},
	}
	topo := BuildTopology(cfg.Services.Names, false, "", map[string]string{"api": "shop", "db": "shop"})
	g := NewSpanGenerator(cfg, topo)
	w := NewTraceWriter("/tmp", "test")

	tr := g.GenerateTrace()
	req := w.tracesToOTLP([]*TraceTemplate{tr})
	if ns, ok := resourceAttr(req.ResourceSpans[0], "service.namespace"); !ok || ns != "shop" {
		t.Fatalf("resource service.namespace = %q (ok=%v), want \"shop\"", ns, ok)
	}
}
