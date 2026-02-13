package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/traces"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/metrics"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/logs"
	"gopkg.in/yaml.v3"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (required)")
	outputDir := flag.String("output-dir", "", "Output directory (overrides config)")
	jsonOutput := flag.Bool("json", false, "Generate JSON output alongside protobuf for debugging")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadGeneratorConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override output directory if specified
	if *outputDir != "" {
		cfg.Output.Directory = *outputDir
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Telemetry Generator")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Configuration: %s\n", *configPath)
	fmt.Printf("Output directory: %s\n", cfg.Output.Directory)
	fmt.Printf("Output prefix: %s\n", cfg.Output.Prefix)
	fmt.Printf("JSON output: %v\n", *jsonOutput)

	// Show estimated sender memory usage
	estimatedMemory := cfg.EstimateMemoryUsage()
	memoryGB := float64(estimatedMemory) / (1024 * 1024 * 1024)
	fmt.Printf("Estimated sender memory: %.2f GB\n", memoryGB)
	if memoryGB > 8.0 {
		fmt.Printf("  ⚠️  High memory usage - consider reducing dataset size\n")
	}
	fmt.Println()

	startTime := time.Now()

	// Generate traces
	if cfg.Traces.Count > 0 {
		fmt.Println("───────────────────────────────────────────────────────────")
		traceGen := traces.NewGenerator(&cfg.Traces, cfg.Output.Directory, cfg.Output.Prefix)
		if err := traceGen.Generate(*jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating traces: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}

	// Generate metrics
	if cfg.Metrics.MetricCount > 0 {
		fmt.Println("───────────────────────────────────────────────────────────")
		metricGen := metrics.NewGenerator(&cfg.Metrics, cfg.Output.Directory, cfg.Output.Prefix)
		if err := metricGen.Generate(*jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating metrics: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}

	// Generate logs
	if cfg.Logs.Count > 0 {
		fmt.Println("───────────────────────────────────────────────────────────")
		logGen := logs.NewGenerator(&cfg.Logs, cfg.Output.Directory, cfg.Output.Prefix)
		if err := logGen.Generate(*jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating logs: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}

	// Write metadata file
	if err := writeMetadata(cfg, cfg.Output.Directory, cfg.Output.Prefix, startTime); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write metadata: %v\n", err)
	}

	elapsed := time.Since(startTime)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("✓ Generation complete in %s\n", elapsed.Round(time.Millisecond))
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// Metadata represents generation metadata
type Metadata struct {
	GeneratedAt  string                    `yaml:"generated_at"`
	Duration     string                    `yaml:"duration"`
	Configuration map[string]interface{}   `yaml:"configuration"`
	Files        map[string]string         `yaml:"files"`
}

// writeMetadata writes a metadata YAML file with generation information
func writeMetadata(cfg *config.GeneratorConfig, outputDir, prefix string, startTime time.Time) error {
	files := make(map[string]string)

	if cfg.Traces.Count > 0 {
		files["traces_pb"] = fmt.Sprintf("%s-traces.pb", prefix)
	}
	if cfg.Metrics.MetricCount > 0 {
		files["metrics_pb"] = fmt.Sprintf("%s-metrics.pb", prefix)
	}
	if cfg.Logs.Count > 0 {
		files["logs_pb"] = fmt.Sprintf("%s-logs.pb", prefix)
	}

	metadata := Metadata{
		GeneratedAt: startTime.Format(time.RFC3339),
		Duration:    time.Since(startTime).Round(time.Millisecond).String(),
		Configuration: map[string]interface{}{
			"traces": map[string]interface{}{
				"count":         cfg.Traces.Count,
				"avg_spans":     cfg.Traces.Spans.AvgPerTrace,
				"services":      cfg.Traces.Services.Count,
			},
			"metrics": map[string]interface{}{
				"count":             cfg.Metrics.MetricCount,
				"timeseries_range":  fmt.Sprintf("%d-%d", cfg.Metrics.TimeSeriesPerMetric.Min, cfg.Metrics.TimeSeriesPerMetric.Max),
			},
			"logs": map[string]interface{}{
				"count": cfg.Logs.Count,
			},
		},
		Files: files,
	}

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	metadataPath := fmt.Sprintf("%s/%s-metadata.yaml", outputDir, prefix)
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Wrote metadata to %s\n", metadataPath)
	return nil
}
