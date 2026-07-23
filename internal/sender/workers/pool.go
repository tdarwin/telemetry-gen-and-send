package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

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

	// scheduler exports late spans (those carrying _template.emit_delay_ms)
	// after the rest of their trace. nil when there is no trace exporter.
	scheduler *deferredScheduler

	// Worker distribution by signal type (exported for visibility)
	TraceWorkers   int
	MetricsWorkers int
	LogsWorkers    int
}

// DeferredOptions configures the deferred-emission scheduler for late spans.
type DeferredOptions struct {
	MaxPending   int
	DrainTimeout time.Duration
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
	deferredOpts DeferredOptions,
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

	// Only traces use deferred emission, so the scheduler needs a trace exporter.
	if traceExporter != nil {
		pool.scheduler = newDeferredScheduler(traceExporter, rateLimiter, reporter, deferredOpts.MaxPending, deferredOpts.DrainTimeout)
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

	// Start the deferred-emission scheduler (for late spans) before workers so
	// it is ready to receive enqueues.
	if p.scheduler != nil {
		p.scheduler.Start()
	}

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

	// Workers are done, so no more deferred spans will be enqueued. Drain the
	// scheduler (honoring each late span's scheduled time up to the drain
	// timeout) before returning, so the process doesn't exit — and the trace
	// exporter isn't closed — while late roots are still pending.
	if p.scheduler != nil {
		if dropped := p.scheduler.Close(); dropped > 0 {
			fmt.Printf("WARNING: %d deferred span(s) were dropped (queue full or drain timeout exceeded)\n", dropped)
		}
	}

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

// deferredReq is a transformed trace payload to be exported after delayMs.
type deferredReq struct {
	delayMs   int64
	req       *otlpcollectortrace.ExportTraceServiceRequest
	spanCount int
}

// sendTraces sends traces in batches based on configured batch size and span
// count limits. Each trace is transformed once (ID regeneration + timestamp
// injection over all of its spans) and then split into an immediate set and any
// deferred (late) spans, so the trace ID stays consistent across every part.
// Large traces are automatically split across multiple batches.
func (p *WorkerPool) sendTraces(ctx context.Context) error {
	if p.templates.Traces == nil || len(p.templates.Traces.ResourceSpans) == 0 {
		return nil
	}

	const maxSpansPerBatch = 10000 // Limit to prevent gRPC message size issues

	totalResourceSpans := len(p.templates.Traces.ResourceSpans)
	currentBatch := make([]*otlptrace.ResourceSpans, 0, p.batchSizeTraces)
	currentSpanCount := 0

	flush := func() error {
		if len(currentBatch) == 0 {
			return nil
		}
		if err := p.sendRawTraceBatch(ctx, currentBatch); err != nil {
			return err
		}
		currentBatch = currentBatch[:0]
		currentSpanCount = 0
		return nil
	}

	for i := 0; i < totalResourceSpans; i++ {
		// Check context periodically
		if i%100 == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		// Transform the whole trace once, then partition into immediate and
		// deferred (late) spans that share the regenerated trace ID.
		immediate, deferred, immSpanCount := p.transformTrace(p.templates.Traces.ResourceSpans[i])

		// Schedule any late spans (e.g. a delayed root) for later export.
		for _, d := range deferred {
			p.enqueueDeferred(d)
		}

		if immSpanCount == 0 {
			// Entire trace is deferred (e.g. a single delayed root span).
			continue
		}

		// If the immediate portion alone exceeds the per-batch span limit,
		// flush pending work and chunk it across batches.
		if immSpanCount > maxSpansPerBatch {
			if err := flush(); err != nil {
				return err
			}
			if err := p.sendLargeImmediate(ctx, immediate, maxSpansPerBatch); err != nil {
				return err
			}
			continue
		}

		wouldExceedSpanLimit := currentSpanCount+immSpanCount > maxSpansPerBatch
		wouldExceedTraceLimit := len(currentBatch) >= p.batchSizeTraces
		if len(currentBatch) > 0 && (wouldExceedSpanLimit || wouldExceedTraceLimit) {
			if err := flush(); err != nil {
				return err
			}
		}

		currentBatch = append(currentBatch, immediate)
		currentSpanCount += immSpanCount
	}

	// Send remaining batch
	return flush()
}

// transformTrace clones a single trace's ResourceSpans, regenerates its IDs and
// injects timestamps over ALL of its spans at once (so the trace ID and span
// IDs stay consistent), then partitions the result into an immediate
// ResourceSpans and zero or more deferred payloads keyed by emit delay.
//
// Note: the generator writes each trace as one ResourceSpans (usually with a
// single ScopeSpans); this function does not split a trace's spans across the
// ID-regeneration pass, which is what keeps a phantom/late root correct.
func (p *WorkerPool) transformTrace(rs *otlptrace.ResourceSpans) (immediate *otlptrace.ResourceSpans, deferred []deferredReq, immSpanCount int) {
	clone := cloneTraceBatch([]*otlptrace.ResourceSpans{rs}).ResourceSpans[0]

	// One-shot ID regeneration across every span in the trace.
	var allSpans []*otlptrace.Span
	for _, ss := range clone.ScopeSpans {
		allSpans = append(allSpans, ss.Spans...)
	}
	p.idRegenerator.RegenerateTraceIDs(allSpans)

	// Capture emit delays BEFORE timestamp injection strips the template attr.
	delays := make(map[*otlptrace.Span]int64)
	for _, sp := range allSpans {
		if d := p.timestampInjector.ExtractEmitDelayMs(sp); d > 0 {
			delays[sp] = d
		}
	}
	p.timestampInjector.InjectSpanTimestamps(allSpans)

	immediate = &otlptrace.ResourceSpans{
		Resource:  clone.Resource,
		SchemaUrl: clone.SchemaUrl,
	}

	for _, ss := range clone.ScopeSpans {
		var immSpans []*otlptrace.Span
		byDelay := make(map[int64][]*otlptrace.Span)
		for _, sp := range ss.Spans {
			if d, ok := delays[sp]; ok {
				byDelay[d] = append(byDelay[d], sp)
			} else {
				immSpans = append(immSpans, sp)
			}
		}

		if len(immSpans) > 0 {
			immediate.ScopeSpans = append(immediate.ScopeSpans, &otlptrace.ScopeSpans{
				Scope:     ss.Scope,
				SchemaUrl: ss.SchemaUrl,
				Spans:     immSpans,
			})
			immSpanCount += len(immSpans)
		}

		for d, spans := range byDelay {
			deferred = append(deferred, deferredReq{
				delayMs:   d,
				spanCount: len(spans),
				req: &otlpcollectortrace.ExportTraceServiceRequest{
					ResourceSpans: []*otlptrace.ResourceSpans{
						{
							Resource:  clone.Resource,
							SchemaUrl: clone.SchemaUrl,
							ScopeSpans: []*otlptrace.ScopeSpans{
								{Scope: ss.Scope, SchemaUrl: ss.SchemaUrl, Spans: spans},
							},
						},
					},
				},
			})
		}
	}

	return immediate, deferred, immSpanCount
}

// enqueueDeferred schedules a late payload for export at now+delay.
func (p *WorkerPool) enqueueDeferred(d deferredReq) {
	if p.scheduler == nil {
		return
	}
	sendAt := time.Now().Add(time.Duration(d.delayMs) * time.Millisecond)
	p.scheduler.Enqueue(d.req, sendAt, d.spanCount)
}

// sendLargeImmediate splits an already-transformed trace with many immediate
// spans across multiple batches. All chunks share the trace's single
// (regenerated) trace ID.
func (p *WorkerPool) sendLargeImmediate(ctx context.Context, rs *otlptrace.ResourceSpans, maxSpansPerBatch int) error {
	for _, ss := range rs.ScopeSpans {
		totalSpans := len(ss.Spans)
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

			if err := p.sendRawTraceBatch(ctx, []*otlptrace.ResourceSpans{chunkRS}); err != nil {
				return err
			}
		}
	}

	return nil
}

// sendRawTraceBatch rate-limits and exports an already-transformed batch of
// resource spans. It performs no cloning, ID regeneration, or timestamp
// injection — that work happens once per trace in transformTrace.
func (p *WorkerPool) sendRawTraceBatch(ctx context.Context, batchResourceSpans []*otlptrace.ResourceSpans) error {
	spanCount := 0
	for _, rs := range batchResourceSpans {
		for _, ss := range rs.ScopeSpans {
			spanCount += len(ss.Spans)
		}
	}
	if spanCount == 0 {
		return nil
	}

	if err := p.rateLimiter.Wait(ctx, spanCount); err != nil {
		return err
	}

	request := &otlpcollectortrace.ExportTraceServiceRequest{ResourceSpans: batchResourceSpans}
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
