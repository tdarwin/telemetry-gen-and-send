package traces

import (
	"strings"
	"testing"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
)

func testTopology() *ServiceTopology {
	return BuildTopology([]string{"api", "worker", "db"}, false, "", nil)
}

func collect(tr *TraceTemplate) []*SpanNode {
	return tr.CollectSpans()
}

// TestFatSpansEverySpan verifies that fat-span mode puts a deterministic number
// of large attributes on every span.
func TestFatSpansEverySpan(t *testing.T) {
	cfg := &config.TracesConfig{
		Count:    5,
		Spans:    config.SpansConfig{AvgPerTrace: 6, StdDev: 0},
		Services: config.ServicesConfig{Count: 3, Names: []string{"api", "worker", "db"}},
		CustomAttributes: config.CustomAttributesConfig{
			PerSpanMin: 3, PerSpanMax: 3, ValueBytes: 200, KeyPrefix: "custom.fat",
		},
	}
	g := NewSpanGenerator(cfg, testTopology())

	for i := 0; i < 10; i++ {
		tr := g.GenerateTrace()
		for _, s := range collect(tr) {
			fat := 0
			for _, a := range s.Attributes {
				if strings.HasPrefix(a.Key, "custom.fat.") {
					fat++
					if got := len(a.Value.GetStringValue()); got != 200 {
						t.Fatalf("fat attr value length = %d, want 200", got)
					}
				}
			}
			if fat != 3 {
				t.Fatalf("fat attrs on span = %d, want 3", fat)
			}
		}
	}
}

// TestBaselineUnchanged verifies that with no new knobs set, spans carry no fat
// attributes, roots have no parent, and no emit delay is stamped.
func TestBaselineUnchanged(t *testing.T) {
	cfg := &config.TracesConfig{
		Count:            5,
		Spans:            config.SpansConfig{AvgPerTrace: 6, StdDev: 0},
		Services:         config.ServicesConfig{Count: 3, Names: []string{"api", "worker", "db"}},
		CustomAttributes: config.CustomAttributesConfig{Count: 5},
	}
	g := NewSpanGenerator(cfg, testTopology())

	for i := 0; i < 20; i++ {
		tr := g.GenerateTrace()
		if tr.RootSpan.ParentID != nil {
			t.Fatalf("baseline root should have nil parent, got %x", tr.RootSpan.ParentID)
		}
		for _, s := range collect(tr) {
			if s.EmitDelayMs != 0 {
				t.Fatalf("baseline span should have no emit delay, got %d", s.EmitDelayMs)
			}
			for _, a := range s.Attributes {
				if strings.HasPrefix(a.Key, "custom.fat.") {
					t.Fatalf("baseline span should have no fat attributes")
				}
			}
		}
	}
}

// TestRootlessAndLateRoot verifies that at 100% the root is orphaned with a
// phantom parent and stamped with the configured emit delay.
func TestRootlessAndLateRoot(t *testing.T) {
	cfg := &config.TracesConfig{
		Count:    5,
		Spans:    config.SpansConfig{AvgPerTrace: 6, StdDev: 0},
		Services: config.ServicesConfig{Count: 3, Names: []string{"api", "worker", "db"}},
		Root: config.RootConfig{
			Rootless: config.RootlessConfig{Enabled: true, Percentage: 100},
			LateRoot: config.LateRootConfig{Enabled: true, Percentage: 100, DelayMs: 75000},
		},
	}
	g := NewSpanGenerator(cfg, testTopology())

	for i := 0; i < 20; i++ {
		tr := g.GenerateTrace()

		// Phantom parent: non-empty and not equal to any real span ID.
		if len(tr.RootSpan.ParentID) == 0 {
			t.Fatalf("rootless root should have a phantom (non-empty) parent")
		}
		ids := collectSpanIDSet(tr)
		if _, collides := ids[string(tr.RootSpan.ParentID)]; collides {
			t.Fatalf("phantom parent collides with a real span ID")
		}
		if tr.RootSpan.EmitDelayMs != 75000 {
			t.Fatalf("late root emit delay = %d, want 75000", tr.RootSpan.EmitDelayMs)
		}
	}
}

// TestHighSpanTraceGetsRootTreatment verifies gigatraces also receive the
// missing/late-root treatment.
func TestHighSpanTraceGetsRootTreatment(t *testing.T) {
	cfg := &config.TracesConfig{
		Count:    1,
		Spans:    config.SpansConfig{AvgPerTrace: 6, StdDev: 0},
		Services: config.ServicesConfig{Count: 3, Names: []string{"api", "worker", "db"}},
		Root: config.RootConfig{
			Rootless: config.RootlessConfig{Enabled: true, Percentage: 100},
		},
	}
	g := NewSpanGenerator(cfg, testTopology())

	tr := g.GenerateHighSpanTrace(500)
	if len(tr.RootSpan.ParentID) == 0 {
		t.Fatalf("high-span-trace root should have a phantom parent when rootless is enabled")
	}
}
