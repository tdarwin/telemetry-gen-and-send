package traces

import (
	"fmt"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
)

// Generator is the main trace generator
type Generator struct {
	config       *config.TracesConfig
	topology     *ServiceTopology
	spanGen      *SpanGenerator
	writer       *TraceWriter
}

// NewGenerator creates a new trace generator
func NewGenerator(cfg *config.TracesConfig, outputDir, prefix string) *Generator {
	topology := BuildTopology(
		cfg.Services.Names,
		cfg.Services.Ingress.Single,
		cfg.Services.Ingress.Service,
	)

	spanGen := NewSpanGenerator(cfg, topology)
	writer := NewTraceWriter(outputDir, prefix)

	return &Generator{
		config:   cfg,
		topology: topology,
		spanGen:  spanGen,
		writer:   writer,
	}
}

// Generate generates all traces according to configuration
func (g *Generator) Generate(writeJSON bool) error {
	fmt.Println("Generating traces...")
	fmt.Printf("  Target trace count: %d\n", g.config.Count)
	fmt.Printf("  Avg spans per trace: %d (±%d)\n",
		g.config.Spans.AvgPerTrace, g.config.Spans.StdDev)
	fmt.Printf("  Services: %d\n", g.config.Services.Count)

	traces := make([]*TraceTemplate, 0, g.config.Count)

	// Generate normal traces
	for i := 0; i < g.config.Count; i++ {
		trace := g.spanGen.GenerateTrace()
		traces = append(traces, trace)

		if (i+1)%1000 == 0 {
			fmt.Printf("  Generated %d/%d traces\n", i+1, g.config.Count)
		}
	}

	// Generate high span count traces if enabled
	if g.config.Spans.HighSpanTraces.Enabled {
		fmt.Printf("Generating %d high span count traces (%d spans each)...\n",
			g.config.Spans.HighSpanTraces.Count,
			g.config.Spans.HighSpanTraces.SpanCount)

		for i := 0; i < g.config.Spans.HighSpanTraces.Count; i++ {
			trace := g.spanGen.GenerateHighSpanTrace(g.config.Spans.HighSpanTraces.SpanCount)
			traces = append(traces, trace)
		}
	}

	// Calculate and print statistics
	stats := CalculateStats(traces)
	fmt.Printf("\nTrace Generation Statistics:\n")
	fmt.Printf("  Total traces: %d\n", stats.TotalTraces)
	fmt.Printf("  Total spans: %d\n", stats.TotalSpans)
	fmt.Printf("  Avg spans/trace: %.2f\n", stats.AvgSpans)
	fmt.Printf("  Min spans: %d\n", stats.MinSpans)
	fmt.Printf("  Max spans: %d\n", stats.MaxSpans)

	// Write to disk
	fmt.Println("\nWriting traces to disk...")
	if err := g.writer.WriteTraces(traces, writeJSON); err != nil {
		return fmt.Errorf("failed to write traces: %w", err)
	}

	fmt.Println("✓ Trace generation complete")
	return nil
}

// GetTopology returns the service topology (useful for debugging)
func (g *Generator) GetTopology() *ServiceTopology {
	return g.topology
}
