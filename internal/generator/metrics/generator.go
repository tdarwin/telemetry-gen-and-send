package metrics

import (
	"fmt"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
)

// Generator is the main metrics generator
type Generator struct {
	config *config.MetricsConfig
	dimGen *DimensionGenerator
	writer *MetricsWriter
}

// NewGenerator creates a new metrics generator
func NewGenerator(cfg *config.MetricsConfig, outputDir, prefix string) *Generator {
	return &Generator{
		config: cfg,
		dimGen: NewDimensionGenerator(),
		writer: NewMetricsWriter(outputDir, prefix),
	}
}

// Generate generates all metrics according to configuration
func (g *Generator) Generate(writeJSON bool) error {
	fmt.Println("Generating metrics...")
	fmt.Printf("  Target metric count: %d\n", g.config.MetricCount)
	fmt.Printf("  Time series per metric: %d-%d (default: %d)\n",
		g.config.TimeSeriesPerMetric.Min,
		g.config.TimeSeriesPerMetric.Max,
		g.config.TimeSeriesPerMetric.Default)

	// Get all available metrics from all types
	allMetrics := GetAllAvailableMetrics()
	fmt.Printf("  Available metrics: %d\n", len(allMetrics))

	// Select metrics to generate (randomly selected from all available)
	selectedMetrics := SelectMetrics(allMetrics, g.config.MetricCount)
	fmt.Printf("  Selected metrics: %d\n", len(selectedMetrics))

	// Generate dimension sets for each metric
	metricTemplates := make([]*MetricTemplate, 0, len(selectedMetrics))
	totalTimeSeries := 0

	for i, metricDef := range selectedMetrics {
		// Determine number of time series for this metric
		timeSeriesCount := g.determineTimeSeriesCount()

		// Generate dimension sets
		dimSets := g.dimGen.GenerateDimensionSets(metricDef, timeSeriesCount)

		template := &MetricTemplate{
			Definition:    metricDef,
			DimensionSets: dimSets,
		}

		metricTemplates = append(metricTemplates, template)
		totalTimeSeries += len(dimSets)

		if (i+1)%100 == 0 {
			fmt.Printf("  Generated %d/%d metrics\n", i+1, len(selectedMetrics))
		}
	}

	// Print statistics
	fmt.Printf("\nMetrics Generation Statistics:\n")
	fmt.Printf("  Total metrics: %d\n", len(metricTemplates))
	fmt.Printf("  Total time series: %d\n", totalTimeSeries)
	fmt.Printf("  Avg time series per metric: %.2f\n",
		float64(totalTimeSeries)/float64(len(metricTemplates)))

	// Write to disk
	fmt.Println("\nWriting metrics to disk...")
	if err := g.writer.WriteMetrics(metricTemplates, writeJSON); err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}

	fmt.Println("✓ Metrics generation complete")
	return nil
}

// determineTimeSeriesCount determines the number of time series for a metric
func (g *Generator) determineTimeSeriesCount() int {
	min := g.config.TimeSeriesPerMetric.Min
	max := g.config.TimeSeriesPerMetric.Max
	defaultCount := g.config.TimeSeriesPerMetric.Default

	// Use default if within range
	if defaultCount >= min && defaultCount <= max {
		// Add some variance around the default
		variance := common.RandomInt(-50, 50)
		count := defaultCount + variance

		// Clamp to min/max
		if count < min {
			count = min
		}
		if count > max {
			count = max
		}

		return count
	}

	// Otherwise pick random in range
	return common.RandomInt(min, max)
}
