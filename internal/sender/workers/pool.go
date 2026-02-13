package workers

import (
	"context"
	"fmt"
	"sync"

	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/exporter"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/loader"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/ratelimit"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/stats"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/transformer"
	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpcollectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	otlplogs "go.opentelemetry.io/proto/otlp/logs/v1"
)

// WorkerPool manages concurrent workers for sending telemetry
type WorkerPool struct {
	numWorkers        int
	templates         *loader.Templates
	traceExporter     *exporter.TraceExporter
	metricsExporter   *exporter.MetricsExporter
	logsExporter      *exporter.LogsExporter
	timestampInjector *transformer.TimestampInjector
	idRegenerator     *transformer.IDRegenerator
	rateLimiter       *ratelimit.Limiter
	reporter          *stats.Reporter
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	numWorkers int,
	templates *loader.Templates,
	traceExporter *exporter.TraceExporter,
	metricsExporter *exporter.MetricsExporter,
	logsExporter *exporter.LogsExporter,
	timestampInjector *transformer.TimestampInjector,
	idRegenerator *transformer.IDRegenerator,
	rateLimiter *ratelimit.Limiter,
	reporter *stats.Reporter,
) *WorkerPool {
	return &WorkerPool{
		numWorkers:        numWorkers,
		templates:         templates,
		traceExporter:     traceExporter,
		metricsExporter:   metricsExporter,
		logsExporter:      logsExporter,
		timestampInjector: timestampInjector,
		idRegenerator:     idRegenerator,
		rateLimiter:       rateLimiter,
		reporter:          reporter,
	}
}

// Run starts the worker pool and sends telemetry for the specified number of iterations
func (p *WorkerPool) Run(ctx context.Context, multiplier int) error {
	// Calculate total iterations
	totalIterations := multiplier
	if multiplier == 0 {
		totalIterations = -1 // Infinite
	}

	// Create work channel
	workCh := make(chan int, p.numWorkers*2) // Buffer for smooth operation

	// WaitGroup to track workers
	var wg sync.WaitGroup

	// Error channel
	errCh := make(chan error, p.numWorkers)

	// Start workers
	for i := 0; i < p.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.worker(ctx, workerID, workCh, errCh)
		}(i)
	}

	// Feed work to workers
	go func() {
		iteration := 0
		for {
			if totalIterations > 0 && iteration >= totalIterations {
				break
			}

			select {
			case <-ctx.Done():
				close(workCh)
				return
			case workCh <- iteration:
				iteration++
				if totalIterations < 0 {
					// Infinite mode, reset counter to prevent overflow
					if iteration > 1000000 {
						iteration = 0
					}
				}
			}
		}
		close(workCh)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// worker is the main worker goroutine
func (p *WorkerPool) worker(ctx context.Context, workerID int, workCh <-chan int, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case iteration, ok := <-workCh:
			if !ok {
				return
			}

			// Send traces
			if p.traceExporter != nil && p.templates.Traces != nil {
				if err := p.sendTraces(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					fmt.Printf("Worker %d error sending traces (iteration %d): %v\n", workerID, iteration, err)
					p.reporter.RecordError()
				}
			}

			// Send metrics
			if p.metricsExporter != nil && p.templates.Metrics != nil {
				if err := p.sendMetrics(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					fmt.Printf("Worker %d error sending metrics (iteration %d): %v\n", workerID, iteration, err)
					p.reporter.RecordError()
				}
			}

			// Send logs
			if p.logsExporter != nil && p.templates.Logs != nil {
				if err := p.sendLogs(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					fmt.Printf("Worker %d error sending logs (iteration %d): %v\n", workerID, iteration, err)
					p.reporter.RecordError()
				}
			}
		}
	}
}

// sendTraces sends a batch of traces
func (p *WorkerPool) sendTraces(ctx context.Context) error {
	// Deep copy the request so we don't modify the template
	request := cloneTraceRequest(p.templates.Traces)

	// Transform: regenerate IDs and inject timestamps
	spanCount := 0
	for _, rs := range request.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			p.idRegenerator.RegenerateTraceIDs(ss.Spans)
			p.timestampInjector.InjectSpanTimestamps(ss.Spans)
			spanCount += len(ss.Spans)
		}
	}

	// Rate limit
	if err := p.rateLimiter.Wait(ctx, spanCount); err != nil {
		return err
	}

	// Export
	if err := p.traceExporter.Export(ctx, request); err != nil {
		return err
	}

	p.reporter.RecordTraces(spanCount)
	return nil
}

// sendMetrics sends a batch of metrics
func (p *WorkerPool) sendMetrics(ctx context.Context) error {
	// Deep copy the request
	request := cloneMetricsRequest(p.templates.Metrics)

	// Transform: inject timestamps
	dataPointCount := 0
	for _, rm := range request.ResourceMetrics {
		for _, sm := range rm.ScopeMetrics {
			for _, metric := range sm.Metrics {
				p.timestampInjector.InjectMetricTimestamps(metric)
				dataPointCount += countMetricDataPoints(metric)
			}
		}
	}

	// Rate limit
	if err := p.rateLimiter.Wait(ctx, dataPointCount); err != nil {
		return err
	}

	// Export
	if err := p.metricsExporter.Export(ctx, request); err != nil {
		return err
	}

	p.reporter.RecordMetrics(dataPointCount)
	return nil
}

// sendLogs sends a batch of logs
func (p *WorkerPool) sendLogs(ctx context.Context) error {
	// Deep copy the request
	request := cloneLogsRequest(p.templates.Logs)

	// Transform: inject timestamps
	logCount := 0
	for _, rl := range request.ResourceLogs {
		for _, sl := range rl.ScopeLogs {
			p.timestampInjector.InjectLogTimestamps(sl.LogRecords)
			logCount += len(sl.LogRecords)
		}
	}

	// Rate limit
	if err := p.rateLimiter.Wait(ctx, logCount); err != nil {
		return err
	}

	// Export
	if err := p.logsExporter.Export(ctx, request); err != nil {
		return err
	}

	p.reporter.RecordLogs(logCount)
	return nil
}

// Helper functions for deep copying protobuf messages
func cloneTraceRequest(src *otlpcollectortrace.ExportTraceServiceRequest) *otlpcollectortrace.ExportTraceServiceRequest {
	if src == nil {
		return nil
	}

	// Deep copy resource spans
	resourceSpans := make([]*otlptrace.ResourceSpans, len(src.ResourceSpans))
	for i, rs := range src.ResourceSpans {
		resourceSpans[i] = &otlptrace.ResourceSpans{
			Resource: rs.Resource, // Resource is immutable, can share
			ScopeSpans: make([]*otlptrace.ScopeSpans, len(rs.ScopeSpans)),
			SchemaUrl: rs.SchemaUrl,
		}

		for j, ss := range rs.ScopeSpans {
			// Deep copy spans array
			spans := make([]*otlptrace.Span, len(ss.Spans))
			for k, span := range ss.Spans {
				// Deep copy each span
				spans[k] = &otlptrace.Span{
					TraceId:           append([]byte(nil), span.TraceId...),
					SpanId:            append([]byte(nil), span.SpanId...),
					TraceState:        span.TraceState,
					ParentSpanId:      append([]byte(nil), span.ParentSpanId...),
					Name:              span.Name,
					Kind:              span.Kind,
					StartTimeUnixNano: span.StartTimeUnixNano,
					EndTimeUnixNano:   span.EndTimeUnixNano,
					Attributes:        span.Attributes, // Attributes are immutable
					DroppedAttributesCount: span.DroppedAttributesCount,
					Events:            span.Events,
					DroppedEventsCount: span.DroppedEventsCount,
					Links:             span.Links,
					DroppedLinksCount: span.DroppedLinksCount,
					Status:            span.Status,
				}
			}

			resourceSpans[i].ScopeSpans[j] = &otlptrace.ScopeSpans{
				Scope:     ss.Scope, // Scope is immutable
				Spans:     spans,
				SchemaUrl: ss.SchemaUrl,
			}
		}
	}

	return &otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: resourceSpans,
	}
}

func cloneMetricsRequest(src *otlpcollectormetrics.ExportMetricsServiceRequest) *otlpcollectormetrics.ExportMetricsServiceRequest {
	if src == nil {
		return nil
	}

	// Deep copy resource metrics
	resourceMetrics := make([]*otlpmetrics.ResourceMetrics, len(src.ResourceMetrics))
	for i, rm := range src.ResourceMetrics {
		resourceMetrics[i] = &otlpmetrics.ResourceMetrics{
			Resource: rm.Resource, // Resource is immutable
			ScopeMetrics: make([]*otlpmetrics.ScopeMetrics, len(rm.ScopeMetrics)),
			SchemaUrl: rm.SchemaUrl,
		}

		for j, sm := range rm.ScopeMetrics {
			// Deep copy metrics array
			metrics := make([]*otlpmetrics.Metric, len(sm.Metrics))
			for k, metric := range sm.Metrics {
				// Deep copy each metric - the data points will be copied by value
				metrics[k] = &otlpmetrics.Metric{
					Name:        metric.Name,
					Description: metric.Description,
					Unit:        metric.Unit,
					Data:        metric.Data, // This contains the data points
				}
			}

			resourceMetrics[i].ScopeMetrics[j] = &otlpmetrics.ScopeMetrics{
				Scope:     sm.Scope, // Scope is immutable
				Metrics:   metrics,
				SchemaUrl: sm.SchemaUrl,
			}
		}
	}

	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: resourceMetrics,
	}
}

func cloneLogsRequest(src *otlpcollectorlogs.ExportLogsServiceRequest) *otlpcollectorlogs.ExportLogsServiceRequest {
	if src == nil {
		return nil
	}

	// Deep copy resource logs
	resourceLogs := make([]*otlplogs.ResourceLogs, len(src.ResourceLogs))
	for i, rl := range src.ResourceLogs {
		resourceLogs[i] = &otlplogs.ResourceLogs{
			Resource: rl.Resource, // Resource is immutable
			ScopeLogs: make([]*otlplogs.ScopeLogs, len(rl.ScopeLogs)),
			SchemaUrl: rl.SchemaUrl,
		}

		for j, sl := range rl.ScopeLogs {
			// Deep copy log records array
			logRecords := make([]*otlplogs.LogRecord, len(sl.LogRecords))
			for k, lr := range sl.LogRecords {
				// Deep copy each log record
				logRecords[k] = &otlplogs.LogRecord{
					TimeUnixNano:         lr.TimeUnixNano,
					ObservedTimeUnixNano: lr.ObservedTimeUnixNano,
					SeverityNumber:       lr.SeverityNumber,
					SeverityText:         lr.SeverityText,
					Body:                 lr.Body, // AnyValue is immutable
					Attributes:           lr.Attributes, // Attributes are immutable
					DroppedAttributesCount: lr.DroppedAttributesCount,
					Flags:                lr.Flags,
					TraceId:              append([]byte(nil), lr.TraceId...),
					SpanId:               append([]byte(nil), lr.SpanId...),
				}
			}

			resourceLogs[i].ScopeLogs[j] = &otlplogs.ScopeLogs{
				Scope:      sl.Scope, // Scope is immutable
				LogRecords: logRecords,
				SchemaUrl:  sl.SchemaUrl,
			}
		}
	}

	return &otlpcollectorlogs.ExportLogsServiceRequest{
		ResourceLogs: resourceLogs,
	}
}

func countMetricDataPoints(metric *otlpmetrics.Metric) int {
	switch data := metric.Data.(type) {
	case *otlpmetrics.Metric_Gauge:
		return len(data.Gauge.DataPoints)
	case *otlpmetrics.Metric_Sum:
		return len(data.Sum.DataPoints)
	case *otlpmetrics.Metric_Histogram:
		return len(data.Histogram.DataPoints)
	default:
		return 0
	}
}
