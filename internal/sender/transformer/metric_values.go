package transformer

import (
	"math"
	"math/rand"
	"sync"
	"time"

	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"
)

// The generator picks one value per series when it writes the template, so a
// plain replay sends the same numbers every time: gauges are flat lines, and
// cumulative counters and histograms never grow (their INCREASE/RATE is 0).
// MetricEvolver keeps per-series state and moves the values on every send so
// the data looks like a live system:
//
//   - gauges follow a slow sine wave (own phase and period per series) plus noise
//   - monotonic cumulative sums grow by a randomised amount per elapsed second
//   - cumulative histograms record new observations, so count, sum and buckets grow
//
// Cumulative series also keep one start time (the sender's launch) instead of
// a fresh one on every send, so backends see a single continuous counter.
const (
	// A series' template value is treated as roughly this many seconds of
	// accumulated activity when deriving how fast it grows.
	growthBaselineSeconds = 3600.0

	gaugeSwing = 0.25 // sine amplitude, as a fraction of the template value
	gaugeNoise = 0.02 // gaussian noise, as a fraction of the template value
)

// MetricEvolver advances template metric values over time.
type MetricEvolver struct {
	mu      sync.Mutex
	live    *otlpcollectormetrics.ExportMetricsServiceRequest
	metrics []*evolvingMetric
	origin  time.Time
	last    time.Time
	rng     *rand.Rand
}

type evolvingMetric struct {
	metric *otlpmetrics.Metric
	series []*seriesState
}

type seriesState struct {
	base    float64   // template value (gauge/sum) or total observation count (histogram)
	phase   float64   // gauge: radians
	period  float64   // gauge: seconds
	rate    float64   // sum: units per second; histogram: observations per second
	acc     float64   // sum: accumulated growth; histogram: fractional observations carried over
	weights []float64 // histogram: relative bucket frequency taken from the template
}

// NewMetricEvolver deep-copies the template request and prepares per-series
// state. Cumulative start times are fixed at now.
func NewMetricEvolver(template *otlpcollectormetrics.ExportMetricsServiceRequest, now time.Time) *MetricEvolver {
	e := &MetricEvolver{
		live:   proto.Clone(template).(*otlpcollectormetrics.ExportMetricsServiceRequest),
		origin: now,
		last:   now,
		rng:    rand.New(rand.NewSource(now.UnixNano())),
	}
	start := uint64(now.UnixNano())

	for _, rm := range e.live.ResourceMetrics {
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				em := &evolvingMetric{metric: m}
				switch data := m.Data.(type) {
				case *otlpmetrics.Metric_Gauge:
					for _, dp := range data.Gauge.DataPoints {
						em.series = append(em.series, e.newGaugeState(numberValue(dp)))
					}
				case *otlpmetrics.Metric_Sum:
					for _, dp := range data.Sum.DataPoints {
						dp.StartTimeUnixNano = start
						base := numberValue(dp)
						em.series = append(em.series, &seriesState{
							base: base,
							rate: math.Max(base/growthBaselineSeconds, 0.05),
						})
					}
				case *otlpmetrics.Metric_Histogram:
					for _, dp := range data.Histogram.DataPoints {
						dp.StartTimeUnixNano = start
						st := &seriesState{base: float64(dp.Count)}
						st.rate = math.Max(st.base/growthBaselineSeconds, 0.05)
						for _, c := range dp.BucketCounts {
							st.weights = append(st.weights, float64(c))
						}
						em.series = append(em.series, st)
					}
				}
				e.metrics = append(e.metrics, em)
			}
		}
	}
	return e
}

func (e *MetricEvolver) newGaugeState(base float64) *seriesState {
	return &seriesState{
		base:   base,
		phase:  e.rng.Float64() * 2 * math.Pi,
		period: 120 + e.rng.Float64()*480, // 2 to 10 minutes
	}
}

// Next advances every series to the given time and returns an independent deep
// copy of the result, safe to modify (for example to stamp timestamps on).
func (e *MetricEvolver) Next(now time.Time) *otlpcollectormetrics.ExportMetricsServiceRequest {
	e.mu.Lock()
	defer e.mu.Unlock()

	dt := now.Sub(e.last).Seconds()
	if dt < 0 {
		dt = 0
	}
	t := now.Sub(e.origin).Seconds()
	e.last = now

	for _, em := range e.metrics {
		switch data := em.metric.Data.(type) {
		case *otlpmetrics.Metric_Gauge:
			for i, dp := range data.Gauge.DataPoints {
				setNumberValue(dp, e.gaugeValue(em.metric.Unit, em.series[i], t))
			}
		case *otlpmetrics.Metric_Sum:
			for i, dp := range data.Sum.DataPoints {
				st := em.series[i]
				st.acc += st.rate * dt * (0.5 + e.rng.Float64())
				setNumberValue(dp, st.base+math.Floor(st.acc))
			}
		case *otlpmetrics.Metric_Histogram:
			for i, dp := range data.Histogram.DataPoints {
				e.growHistogram(dp, em.series[i], dt)
			}
		}
	}

	return proto.Clone(e.live).(*otlpcollectormetrics.ExportMetricsServiceRequest)
}

func (e *MetricEvolver) gaugeValue(unit string, st *seriesState, t float64) float64 {
	v := st.base * (1 + gaugeSwing*math.Sin(2*math.Pi*t/st.period+st.phase) + gaugeNoise*e.rng.NormFloat64())
	if unit == "%" {
		return math.Min(math.Max(v, 0), 100)
	}
	if st.base >= 0 {
		v = math.Max(v, 0)
	}
	return v
}

func (e *MetricEvolver) growHistogram(dp *otlpmetrics.HistogramDataPoint, st *seriesState, dt float64) {
	if len(dp.BucketCounts) != len(st.weights) || len(dp.ExplicitBounds)+1 != len(dp.BucketCounts) {
		return
	}
	var total float64
	for _, w := range st.weights {
		total += w
	}
	if total == 0 {
		return
	}

	st.acc += st.rate * dt * (0.5 + e.rng.Float64())
	n := math.Floor(st.acc)
	st.acc -= n

	for i, w := range st.weights {
		share := n * w / total
		add := math.Floor(share)
		if e.rng.Float64() < share-add {
			add++
		}
		if add == 0 {
			continue
		}
		dp.BucketCounts[i] += uint64(add)
		dp.Count += uint64(add)
		if dp.Sum != nil {
			*dp.Sum += add * bucketRepresentative(dp.ExplicitBounds, i)
		}
	}
}

// bucketRepresentative returns the value used to account for an observation
// in bucket i: its upper bound, or twice the top bound for the overflow bucket.
func bucketRepresentative(bounds []float64, i int) float64 {
	if i < len(bounds) {
		return bounds[i]
	}
	if len(bounds) == 0 {
		return 0
	}
	return bounds[len(bounds)-1] * 2
}

func numberValue(dp *otlpmetrics.NumberDataPoint) float64 {
	switch v := dp.Value.(type) {
	case *otlpmetrics.NumberDataPoint_AsInt:
		return float64(v.AsInt)
	case *otlpmetrics.NumberDataPoint_AsDouble:
		return v.AsDouble
	}
	return 0
}

// setNumberValue writes v back using the data point's existing value type.
func setNumberValue(dp *otlpmetrics.NumberDataPoint, v float64) {
	if _, ok := dp.Value.(*otlpmetrics.NumberDataPoint_AsInt); ok {
		dp.Value = &otlpmetrics.NumberDataPoint_AsInt{AsInt: int64(math.Round(v))}
		return
	}
	dp.Value = &otlpmetrics.NumberDataPoint_AsDouble{AsDouble: v}
}
