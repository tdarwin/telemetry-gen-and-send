package stats

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Reporter tracks and reports sending statistics
type Reporter struct {
	tracesSent   atomic.Int64
	metricsSent  atomic.Int64
	logsSent     atomic.Int64
	errors       atomic.Int64
	startTime    time.Time
	mu           sync.Mutex
	lastReport   time.Time
	reportTicker *time.Ticker
	stopCh       chan struct{}
}

// NewReporter creates a new stats reporter
func NewReporter() *Reporter {
	return &Reporter{
		startTime:  time.Now(),
		lastReport: time.Now(),
		stopCh:     make(chan struct{}),
	}
}

// RecordTraces records traces sent
func (r *Reporter) RecordTraces(count int) {
	r.tracesSent.Add(int64(count))
}

// RecordMetrics records metrics sent
func (r *Reporter) RecordMetrics(count int) {
	r.metricsSent.Add(int64(count))
}

// RecordLogs records logs sent
func (r *Reporter) RecordLogs(count int) {
	r.logsSent.Add(int64(count))
}

// RecordError records an error
func (r *Reporter) RecordError() {
	r.errors.Add(1)
}

// StartPeriodicReporting starts periodic stat reporting
func (r *Reporter) StartPeriodicReporting(interval time.Duration) {
	r.reportTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-r.reportTicker.C:
				r.PrintStats()
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop stops periodic reporting
func (r *Reporter) Stop() {
	if r.reportTicker != nil {
		r.reportTicker.Stop()
	}
	close(r.stopCh)
}

// PrintStats prints current statistics
func (r *Reporter) PrintStats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.startTime)
	sinceLastReport := now.Sub(r.lastReport)

	traces := r.tracesSent.Load()
	metrics := r.metricsSent.Load()
	logs := r.logsSent.Load()
	errs := r.errors.Load()

	totalEvents := traces + metrics + logs

	// Calculate overall rate
	overallRate := float64(totalEvents) / elapsed.Seconds()

	// Calculate rate since last report
	recentRate := float64(totalEvents) / sinceLastReport.Seconds()

	fmt.Printf("\n[%s] Stats:\n", now.Format("15:04:05"))
	fmt.Printf("  Traces sent: %d\n", traces)
	fmt.Printf("  Metrics sent: %d\n", metrics)
	fmt.Printf("  Logs sent: %d\n", logs)
	fmt.Printf("  Total events: %d\n", totalEvents)
	fmt.Printf("  Errors: %d\n", errs)
	fmt.Printf("  Elapsed: %s\n", elapsed.Round(time.Second))
	fmt.Printf("  Overall rate: %.0f events/sec\n", overallRate)
	fmt.Printf("  Recent rate: %.0f events/sec\n", recentRate)

	r.lastReport = now
}

// GetStats returns current statistics
func (r *Reporter) GetStats() (traces, metrics, logs, errors int64, elapsed time.Duration) {
	return r.tracesSent.Load(),
		r.metricsSent.Load(),
		r.logsSent.Load(),
		r.errors.Load(),
		time.Since(r.startTime)
}

// PrintFinalStats prints final statistics
func (r *Reporter) PrintFinalStats() {
	traces, metrics, logs, errs, elapsed := r.GetStats()
	totalEvents := traces + metrics + logs
	rate := float64(totalEvents) / elapsed.Seconds()

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Final Statistics")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Total traces sent:  %d\n", traces)
	fmt.Printf("Total metrics sent: %d\n", metrics)
	fmt.Printf("Total logs sent:    %d\n", logs)
	fmt.Printf("Total events sent:  %d\n", totalEvents)
	fmt.Printf("Total errors:       %d\n", errs)
	fmt.Printf("Total duration:     %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Average rate:       %.0f events/sec\n", rate)
	fmt.Println("═══════════════════════════════════════════════════════════")
}
