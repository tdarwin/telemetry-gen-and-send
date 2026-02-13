package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/exporter"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/loader"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/ratelimit"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/stats"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/transformer"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/workers"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (required)")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadSenderConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Telemetry Sender")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Configuration: %s\n", *configPath)
	fmt.Printf("OTLP Endpoint: %s\n", cfg.OTLP.Endpoint)
	fmt.Printf("Rate limit: %d events/sec\n", cfg.Sending.RateLimit.EventsPerSecond)
	fmt.Printf("Concurrency: %d workers\n", cfg.Sending.Concurrency)
	fmt.Println()

	// Load templates
	fmt.Println("Loading templates...")
	ldr := loader.NewLoader()
	templates, err := ldr.Load(cfg.Input.Traces, cfg.Input.Metrics, cfg.Input.Logs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	// Initialize exporters
	var traceExporter *exporter.TraceExporter
	var metricsExporter *exporter.MetricsExporter
	var logsExporter *exporter.LogsExporter

	if cfg.HasTraces() && templates.Traces != nil {
		traceExporter, err = exporter.NewTraceExporter(cfg.OTLP.Endpoint, cfg.OTLP.Headers, cfg.OTLP.Insecure)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating trace exporter: %v\n", err)
			os.Exit(1)
		}
		defer traceExporter.Close()
		fmt.Println("✓ Trace exporter initialized")
	}

	if cfg.HasMetrics() && templates.Metrics != nil {
		metricsExporter, err = exporter.NewMetricsExporter(cfg.OTLP.Endpoint, cfg.OTLP.Headers, cfg.OTLP.Insecure)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating metrics exporter: %v\n", err)
			os.Exit(1)
		}
		defer metricsExporter.Close()
		fmt.Println("✓ Metrics exporter initialized")
	}

	if cfg.HasLogs() && templates.Logs != nil {
		logsExporter, err = exporter.NewLogsExporter(cfg.OTLP.Endpoint, cfg.OTLP.Headers, cfg.OTLP.Insecure)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logs exporter: %v\n", err)
			os.Exit(1)
		}
		defer logsExporter.Close()
		fmt.Println("✓ Logs exporter initialized")
	}

	// Initialize transformers
	timestampInjector := transformer.NewTimestampInjector(cfg.Timestamps.JitterMs, cfg.Timestamps.BackdateMs)
	idRegenerator := transformer.NewIDRegenerator()

	// Initialize rate limiter
	rateLimiter := ratelimit.NewLimiter(cfg.Sending.RateLimit.EventsPerSecond)

	// Initialize stats reporter
	reporter := stats.NewReporter()
	reporter.StartPeriodicReporting(5 * time.Second)
	defer reporter.Stop()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle duration limit
	duration, err := cfg.GetDuration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing duration: %v\n", err)
		os.Exit(1)
	}

	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
		fmt.Printf("\nSending for %s...\n", duration)
	} else {
		fmt.Println("\nSending indefinitely (Ctrl+C to stop)...")
	}

	// Handle interrupts
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Create worker pool
	pool := workers.NewWorkerPool(
		cfg.Sending.Concurrency,
		templates,
		traceExporter,
		metricsExporter,
		logsExporter,
		timestampInjector,
		idRegenerator,
		rateLimiter,
		reporter,
	)

	// Start sending
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Starting %d worker(s)...\n", cfg.Sending.Concurrency)
	fmt.Println("Sending telemetry...")
	fmt.Println()

	// Run worker pool
	if err := pool.Run(ctx, cfg.Sending.Multiplier); err != nil {
		fmt.Fprintf(os.Stderr, "Error running worker pool: %v\n", err)
	}

	fmt.Println("\n\nShutting down...")
	reporter.PrintFinalStats()
}
