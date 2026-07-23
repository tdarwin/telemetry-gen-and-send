package workers

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/ratelimit"
	"github.com/honeycomb/telemetry-gen-and-send/internal/sender/stats"
	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

// deferredExportTimeout bounds a single deferred export so a dead endpoint
// cannot hang the scheduler goroutine indefinitely.
const deferredExportTimeout = 30 * time.Second

// traceExportSink is the subset of *exporter.TraceExporter the scheduler needs.
// It exists so the scheduler can be tested without a real gRPC connection.
type traceExportSink interface {
	Export(ctx context.Context, request *otlpcollectortrace.ExportTraceServiceRequest) error
}

// deferredItem is a trace payload scheduled to be exported at sendAt.
type deferredItem struct {
	sendAt    time.Time
	request   *otlpcollectortrace.ExportTraceServiceRequest
	spanCount int
	seq       uint64
}

// itemHeap is a min-heap of deferredItems ordered by sendAt, with seq breaking
// ties so equal timestamps keep FIFO order.
type itemHeap []*deferredItem

func (h itemHeap) Len() int { return len(h) }
func (h itemHeap) Less(i, j int) bool {
	if h[i].sendAt.Equal(h[j].sendAt) {
		return h[i].seq < h[j].seq
	}
	return h[i].sendAt.Before(h[j].sendAt)
}
func (h itemHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *itemHeap) Push(x any)   { *h = append(*h, x.(*deferredItem)) }
func (h *itemHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return it
}

// deferredScheduler exports trace payloads at a scheduled wall-clock time using
// a single goroutine and a single timer. It is used to deliver spans carrying a
// positive _template.emit_delay_ms later than the rest of their trace (e.g. a
// root span that must arrive after the receiver's trace timeout).
//
// Exports are decoupled from the main send context on purpose: an interrupt or
// duration limit that stops the send loop must NOT drop already-scheduled late
// spans — they still drain on their own schedule, bounded by drainTimeout.
type deferredScheduler struct {
	exporter     traceExportSink
	limiter      *ratelimit.Limiter
	reporter     *stats.Reporter
	maxPending   int
	drainTimeout time.Duration

	mu   sync.Mutex
	heap itemHeap
	seq  uint64

	wake     chan struct{}
	closeCh  chan struct{}
	loopDone chan struct{}
	closed   atomic.Bool
	dropped  atomic.Int64
}

func newDeferredScheduler(exp traceExportSink, limiter *ratelimit.Limiter, reporter *stats.Reporter, maxPending int, drainTimeout time.Duration) *deferredScheduler {
	return &deferredScheduler{
		exporter:     exp,
		limiter:      limiter,
		reporter:     reporter,
		maxPending:   maxPending,
		drainTimeout: drainTimeout,
		wake:         make(chan struct{}, 1),
		closeCh:      make(chan struct{}),
		loopDone:     make(chan struct{}),
	}
}

// Start launches the scheduler goroutine.
func (s *deferredScheduler) Start() {
	go s.loop()
}

// Enqueue schedules request for export at sendAt. It returns false (and counts
// the spans as dropped) when the pending queue is full. Callers must not
// Enqueue after Close.
func (s *deferredScheduler) Enqueue(request *otlpcollectortrace.ExportTraceServiceRequest, sendAt time.Time, spanCount int) bool {
	if s.closed.Load() {
		s.dropped.Add(int64(spanCount))
		return false
	}

	s.mu.Lock()
	if s.maxPending > 0 && len(s.heap) >= s.maxPending {
		s.mu.Unlock()
		s.dropped.Add(int64(spanCount))
		return false
	}
	s.seq++
	heap.Push(&s.heap, &deferredItem{
		sendAt:    sendAt,
		request:   request,
		spanCount: spanCount,
		seq:       s.seq,
	})
	s.mu.Unlock()

	// Wake the loop in case this item is sooner than the one it's timing.
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return true
}

// Close stops accepting new items, drains the queue (honoring each item's
// scheduled time up to drainTimeout), and returns the number of spans that were
// dropped over the scheduler's lifetime. It blocks until the loop exits.
func (s *deferredScheduler) Close() int64 {
	s.closed.Store(true)
	close(s.closeCh)
	<-s.loopDone
	return s.dropped.Load()
}

// loop is the single consumer goroutine. It fires matured items and, once
// closed, drains the remainder within drainTimeout.
func (s *deferredScheduler) loop() {
	defer close(s.loopDone)

	closeCh := s.closeCh
	var drainDeadlineCh <-chan time.Time

	// arm transitions the loop into draining mode: stop selecting on closeCh
	// (a closed channel is always ready and would busy-spin) and start the
	// bounded drain deadline.
	arm := func() {
		closeCh = nil
		if s.drainTimeout > 0 {
			drainDeadlineCh = time.After(s.drainTimeout)
		}
	}

	for {
		s.mu.Lock()
		now := time.Now()
		if len(s.heap) == 0 {
			s.mu.Unlock()
			if closeCh == nil {
				// Closed and nothing left to send.
				return
			}
			select {
			case <-s.wake:
			case <-closeCh:
				arm()
			case <-drainDeadlineCh:
				return
			}
			continue
		}

		head := s.heap[0]
		if !head.sendAt.After(now) {
			it := heap.Pop(&s.heap).(*deferredItem)
			s.mu.Unlock()
			s.fire(it)
			continue
		}
		wait := head.sendAt.Sub(now)
		s.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-s.wake:
			timer.Stop()
		case <-closeCh:
			timer.Stop()
			arm()
		case <-drainDeadlineCh:
			timer.Stop()
			s.dropRemaining()
			return
		}
	}
}

// fire exports one deferred payload, rate-limited and counted like any other
// batch. Errors are recorded but never crash the scheduler.
func (s *deferredScheduler) fire(it *deferredItem) {
	ctx, cancel := context.WithTimeout(context.Background(), deferredExportTimeout)
	defer cancel()

	if err := s.limiter.Wait(ctx, it.spanCount); err != nil {
		s.reporter.RecordError()
		return
	}
	if err := s.exporter.Export(ctx, it.request); err != nil {
		s.reporter.RecordError()
		return
	}
	s.reporter.RecordTraces(it.spanCount)
}

// dropRemaining discards everything still queued (drain deadline exceeded) and
// counts it as dropped.
func (s *deferredScheduler) dropRemaining() {
	s.mu.Lock()
	var spans int64
	for _, it := range s.heap {
		spans += int64(it.spanCount)
	}
	s.heap = nil
	s.mu.Unlock()
	s.dropped.Add(spans)
}
