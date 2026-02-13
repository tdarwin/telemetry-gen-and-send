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
// Workers are divided by signal type for realistic load patterns
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
	batchSizeTraces   int
	batchSizeMetrics  int
	batchSizeLogs     int

	// Worker distribution by signal type (exported for visibility)
	TraceWorkers   int
	MetricsWorkers int
	LogsWorkers    int
}

// NewWorkerPool creates a new worker pool with workers divided by signal type
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
	batchSizeTraces int,
	batchSizeMetrics int,
	batchSizeLogs int,
) *WorkerPool {
	pool := &WorkerPool{
		numWorkers:        numWorkers,
		templates:         templates,
		traceExporter:     traceExporter,
		metricsExporter:   metricsExporter,
		logsExporter:      logsExporter,
		timestampInjector: timestampInjector,
		idRegenerator:     idRegenerator,
		rateLimiter:       rateLimiter,
		reporter:          reporter,
		batchSizeTraces:   batchSizeTraces,
		batchSizeMetrics:  batchSizeMetrics,
		batchSizeLogs:     batchSizeLogs,
	}

	// Calculate worker distribution based on data volume
	pool.calculateWorkerDistribution()

	return pool
}

// calculateWorkerDistribution divides workers by signal type based on data volume
// Minimum 1 worker per active signal type, remaining workers distributed by volume
func (p *WorkerPool) calculateWorkerDistribution() {
	// Count total events for each signal type
	var traceEvents, metricEvents, logEvents int64

	if p.templates.Traces != nil {
		for _, rs := range p.templates.Traces.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				traceEvents += int64(len(ss.Spans))
			}
		}
	}

	if p.templates.Metrics != nil {
		for _, rm := range p.templates.Metrics.ResourceMetrics {
			for _, sm := range rm.ScopeMetrics {
				for _, metric := range sm.Metrics {
					metricEvents += int64(countMetricDataPoints(metric))
				}
			}
		}
	}

	if p.templates.Logs != nil {
		for _, rl := range p.templates.Logs.ResourceLogs {
			for _, sl := range rl.ScopeLogs {
				logEvents += int64(len(sl.LogRecords))
			}
		}
	}

	totalEvents := traceEvents + metricEvents + logEvents
	if totalEvents == 0 {
		// No data, assign all workers to traces as fallback
		p.TraceWorkers = p.numWorkers
		return
	}

	// Count active signal types
	activeTypes := 0
	if traceEvents > 0 {
		activeTypes++
	}
	if metricEvents > 0 {
		activeTypes++
	}
	if logEvents > 0 {
		activeTypes++
	}

	// Ensure minimum workers (at least 1 per active type, minimum 3 total)
	minWorkers := activeTypes
	if minWorkers < 3 && p.numWorkers >= 3 {
		minWorkers = 3
	}

	if p.numWorkers < minWorkers {
		// Not enough workers - assign at least 1 to each active type
		if traceEvents > 0 {
			p.TraceWorkers = 1
		}
		if metricEvents > 0 {
			p.MetricsWorkers = 1
		}
		if logEvents > 0 {
			p.LogsWorkers = 1
		}
		return
	}

	// Distribute workers proportionally by event count
	p.TraceWorkers = int(float64(p.numWorkers) * float64(traceEvents) / float64(totalEvents))
	p.MetricsWorkers = int(float64(p.numWorkers) * float64(metricEvents) / float64(totalEvents))
	p.LogsWorkers = int(float64(p.numWorkers) * float64(logEvents) / float64(totalEvents))

	// Ensure each active type gets at least 1 worker
	if traceEvents > 0 && p.TraceWorkers == 0 {
		p.TraceWorkers = 1
	}
	if metricEvents > 0 && p.MetricsWorkers == 0 {
		p.MetricsWorkers = 1
	}
	if logEvents > 0 && p.LogsWorkers == 0 {
		p.LogsWorkers = 1
	}

	// Distribute any remaining workers due to rounding
	assigned := p.TraceWorkers + p.MetricsWorkers + p.LogsWorkers
	remaining := p.numWorkers - assigned

	// Give remaining workers to the type with most events
	if remaining > 0 {
		if traceEvents >= metricEvents && traceEvents >= logEvents {
			p.TraceWorkers += remaining
		} else if metricEvents >= logEvents {
			p.MetricsWorkers += remaining
		} else {
			p.LogsWorkers += remaining
		}
	}
}

// Run starts the worker pool with specialized workers for each signal type
// Workers send their assigned signal type continuously until context is cancelled
func (p *WorkerPool) Run(ctx context.Context, multiplier int) error {
	// WaitGroup to track all workers
	var wg sync.WaitGroup

	// Error channel
	errCh := make(chan error, p.numWorkers)

	// Start trace workers
	if p.TraceWorkers > 0 && p.traceExporter != nil && p.templates.Traces != nil {
		for i := 0; i < p.TraceWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				p.traceWorker(ctx, workerID, multiplier, errCh)
			}(i)
		}
	}

	// Start metrics workers
	if p.MetricsWorkers > 0 && p.metricsExporter != nil && p.templates.Metrics != nil {
		for i := 0; i < p.MetricsWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				p.metricsWorker(ctx, workerID, multiplier, errCh)
			}(i)
		}
	}

	// Start log workers
	if p.LogsWorkers > 0 && p.logsExporter != nil && p.templates.Logs != nil {
		for i := 0; i < p.LogsWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				p.logsWorker(ctx, workerID, multiplier, errCh)
			}(i)
		}
	}

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

// traceWorker continuously sends traces until context is cancelled
func (p *WorkerPool) traceWorker(ctx context.Context, workerID int, multiplier int, errCh chan<- error) {
	iteration := 0
	maxIterations := multiplier
	if multiplier == 0 {
		maxIterations = -1 // Infinite
	}

	for {
		// Check if we've reached max iterations
		if maxIterations > 0 && iteration >= maxIterations {
			return
		}

		// Check context
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Send traces
		if err := p.sendTraces(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Printf("Trace worker %d error (iteration %d): %v\n", workerID, iteration, err)
			p.reporter.RecordError()
		}

		iteration++
		if maxIterations < 0 && iteration > 1000000 {
			iteration = 0 // Prevent overflow in infinite mode
		}
	}
}

// metricsWorker continuously sends metrics until context is cancelled
func (p *WorkerPool) metricsWorker(ctx context.Context, workerID int, multiplier int, errCh chan<- error) {
	iteration := 0
	maxIterations := multiplier
	if multiplier == 0 {
		maxIterations = -1 // Infinite
	}

	for {
		// Check if we've reached max iterations
		if maxIterations > 0 && iteration >= maxIterations {
			return
		}

		// Check context
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Send metrics
		if err := p.sendMetrics(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Printf("Metrics worker %d error (iteration %d): %v\n", workerID, iteration, err)
			p.reporter.RecordError()
		}

		iteration++
		if maxIterations < 0 && iteration > 1000000 {
			iteration = 0 // Prevent overflow in infinite mode
		}
	}
}

// logsWorker continuously sends logs until context is cancelled
func (p *WorkerPool) logsWorker(ctx context.Context, workerID int, multiplier int, errCh chan<- error) {
	iteration := 0
	maxIterations := multiplier
	if multiplier == 0 {
		maxIterations = -1 // Infinite
	}

	for {
		// Check if we've reached max iterations
		if maxIterations > 0 && iteration >= maxIterations {
			return
		}

		// Check context
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Send logs
		if err := p.sendLogs(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Printf("Logs worker %d error (iteration %d): %v\n", workerID, iteration, err)
			p.reporter.RecordError()
		}

		iteration++
		if maxIterations < 0 && iteration > 1000000 {
			iteration = 0 // Prevent overflow in infinite mode
		}
	}
}

// sendTraces sends traces in batches based on configured batch size and span count limits
// Large traces are automatically split across multiple batches
func (p *WorkerPool) sendTraces(ctx context.Context) error {
	if p.templates.Traces == nil || len(p.templates.Traces.ResourceSpans) == 0 {
		return nil
	}

	const maxSpansPerBatch = 10000 // Limit to prevent gRPC message size issues

	totalResourceSpans := len(p.templates.Traces.ResourceSpans)
	currentBatch := make([]*otlptrace.ResourceSpans, 0, p.batchSizeTraces)
	currentSpanCount := 0

	for i := 0; i < totalResourceSpans; i++ {
		// Check context periodically
		if i%100 == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		rs := p.templates.Traces.ResourceSpans[i]

		// Count spans in this resource span
		rsSpanCount := 0
		for _, ss := range rs.ScopeSpans {
			rsSpanCount += len(ss.Spans)
		}

		// If this trace fits in a single batch, handle normally
		if rsSpanCount <= maxSpansPerBatch {
			// Check if adding this trace would exceed limits
			wouldExceedSpanLimit := currentSpanCount+rsSpanCount > maxSpansPerBatch
			wouldExceedTraceLimit := len(currentBatch) >= p.batchSizeTraces

			// Send current batch if adding this trace would exceed limits
			if len(currentBatch) > 0 && (wouldExceedSpanLimit || wouldExceedTraceLimit) {
				if err := p.sendTraceBatch(ctx, currentBatch); err != nil {
					return err
				}
				currentBatch = currentBatch[:0] // Reset slice
				currentSpanCount = 0
			}

			// Add this trace to current batch
			currentBatch = append(currentBatch, rs)
			currentSpanCount += rsSpanCount
		} else {
			// This trace is too large for a single batch - need to split it
			// First, flush any pending batch
			if len(currentBatch) > 0 {
				if err := p.sendTraceBatch(ctx, currentBatch); err != nil {
					return err
				}
				currentBatch = currentBatch[:0]
				currentSpanCount = 0
			}

			// Split this large trace across multiple batches
			if err := p.sendLargeTrace(ctx, rs, maxSpansPerBatch); err != nil {
				return err
			}
		}
	}

	// Send remaining batch
	if len(currentBatch) > 0 {
		if err := p.sendTraceBatch(ctx, currentBatch); err != nil {
			return err
		}
	}

	return nil
}

// sendLargeTrace splits a trace with many spans across multiple batches
func (p *WorkerPool) sendLargeTrace(ctx context.Context, rs *otlptrace.ResourceSpans, maxSpansPerBatch int) error {
	// For each ScopeSpans in this ResourceSpans
	for _, ss := range rs.ScopeSpans {
		totalSpans := len(ss.Spans)

		// Split spans into chunks
		for offset := 0; offset < totalSpans; offset += maxSpansPerBatch {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			end := offset + maxSpansPerBatch
			if end > totalSpans {
				end = totalSpans
			}

			// Create a new ResourceSpans with just this chunk of spans
			chunkRS := &otlptrace.ResourceSpans{
				Resource:  rs.Resource,
				SchemaUrl: rs.SchemaUrl,
				ScopeSpans: []*otlptrace.ScopeSpans{
					{
						Scope:     ss.Scope,
						SchemaUrl: ss.SchemaUrl,
						Spans:     ss.Spans[offset:end],
					},
				},
			}

			// Send this chunk as its own batch
			if err := p.sendTraceBatch(ctx, []*otlptrace.ResourceSpans{chunkRS}); err != nil {
				return err
			}
		}
	}

	return nil
}

// sendTraceBatch sends a single batch of traces
func (p *WorkerPool) sendTraceBatch(ctx context.Context, batchResourceSpans []*otlptrace.ResourceSpans) error {
	// Clone the batch
	request := cloneTraceBatch(batchResourceSpans)

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

// cloneTraceBatch clones a batch of resource spans (memory-efficient batching)
func cloneTraceBatch(srcResourceSpans []*otlptrace.ResourceSpans) *otlpcollectortrace.ExportTraceServiceRequest {
	if srcResourceSpans == nil {
		return nil
	}

	// Deep copy only the specified resource spans
	resourceSpans := make([]*otlptrace.ResourceSpans, len(srcResourceSpans))
	for i, rs := range srcResourceSpans {
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
