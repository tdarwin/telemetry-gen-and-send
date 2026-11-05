package transformer

import (
	"testing"
	"time"

	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func templateRequest() *otlpcollectormetrics.ExportMetricsServiceRequest {
	sum := 0.0
	bounds := []float64{0.1, 1, 10}
	counts := []uint64{100, 300, 500, 100}
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{{
			ScopeMetrics: []*otlpmetrics.ScopeMetrics{{
				Metrics: []*otlpmetrics.Metric{
					{Name: "cpu", Unit: "%", Data: &otlpmetrics.Metric_Gauge{Gauge: &otlpmetrics.Gauge{
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{Value: &otlpmetrics.NumberDataPoint_AsDouble{AsDouble: 40}},
						},
					}}},
					{Name: "requests", Data: &otlpmetrics.Metric_Sum{Sum: &otlpmetrics.Sum{
						IsMonotonic: true,
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{Value: &otlpmetrics.NumberDataPoint_AsInt{AsInt: 36000}},
						},
					}}},
					{Name: "latency", Data: &otlpmetrics.Metric_Histogram{Histogram: &otlpmetrics.Histogram{
						DataPoints: []*otlpmetrics.HistogramDataPoint{
							{Count: 1000, Sum: &sum, BucketCounts: counts, ExplicitBounds: bounds},
						},
					}}},
				},
			}},
		}},
	}
}

func metricsOf(r *otlpcollectormetrics.ExportMetricsServiceRequest) (g, s, h interface{}) {
	ms := r.ResourceMetrics[0].ScopeMetrics[0].Metrics
	return ms[0].GetGauge().DataPoints[0], ms[1].GetSum().DataPoints[0], ms[2].GetHistogram().DataPoints[0]
}

func TestMetricEvolverValuesMove(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0)
	e := NewMetricEvolver(templateRequest(), t0)

	var prevSum int64 = 36000
	var prevHist uint64 = 1000
	gaugeValues := map[float64]bool{}

	for i := 1; i <= 30; i++ {
		r := e.Next(t0.Add(time.Duration(i) * 10 * time.Second))
		g, s, h := metricsOf(r)
		gdp := g.(*otlpmetrics.NumberDataPoint)
		sdp := s.(*otlpmetrics.NumberDataPoint)
		hdp := h.(*otlpmetrics.HistogramDataPoint)

		// Gauge: changes, and a percentage never leaves 0..100.
		v := gdp.GetAsDouble()
		if v < 0 || v > 100 {
			t.Fatalf("step %d: %% gauge out of range: %v", i, v)
		}
		gaugeValues[v] = true

		// Counter: never decreases and keeps its integer type.
		cur := sdp.GetAsInt()
		if cur < prevSum {
			t.Fatalf("step %d: counter went backwards: %d -> %d", i, prevSum, cur)
		}
		prevSum = cur

		// Histogram: count equals the bucket total and never decreases.
		var total uint64
		for _, c := range hdp.BucketCounts {
			total += c
		}
		if hdp.Count != total {
			t.Fatalf("step %d: count %d != bucket total %d", i, hdp.Count, total)
		}
		if hdp.Count < prevHist {
			t.Fatalf("step %d: histogram count went backwards: %d -> %d", i, prevHist, hdp.Count)
		}
		prevHist = hdp.Count

		// Cumulative series keep one start time for the whole run.
		if sdp.StartTimeUnixNano != uint64(t0.UnixNano()) || hdp.StartTimeUnixNano != uint64(t0.UnixNano()) {
			t.Fatalf("step %d: start time changed", i)
		}
	}

	if len(gaugeValues) < 25 {
		t.Errorf("gauge barely moved: %d distinct values in 30 sends", len(gaugeValues))
	}
	// 36000 at 10/s baseline, 300s elapsed, factor averaging 1: ~3000 growth.
	if prevSum < 36000+1500 || prevSum > 36000+4500 {
		t.Errorf("counter growth %d outside expected range for 300s", prevSum-36000)
	}
	// 1000 observations at ~0.28/s baseline over 300s: roughly 80 more.
	if prevHist < 1040 || prevHist > 1200 {
		t.Errorf("histogram count %d outside expected range", prevHist)
	}
}

func TestMetricEvolverReturnsIndependentCopies(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0)
	e := NewMetricEvolver(templateRequest(), t0)
	tpl := templateRequest()

	first := e.Next(t0.Add(10 * time.Second))
	_, s, _ := metricsOf(first)
	s.(*otlpmetrics.NumberDataPoint).TimeUnixNano = 42 // what the timestamp injector does
	s.(*otlpmetrics.NumberDataPoint).Value = &otlpmetrics.NumberDataPoint_AsInt{AsInt: -1}

	second := e.Next(t0.Add(20 * time.Second))
	_, s2, _ := metricsOf(second)
	if got := s2.(*otlpmetrics.NumberDataPoint); got.TimeUnixNano != 0 || got.GetAsInt() < 36000 {
		t.Errorf("mutating a returned request leaked into the next one: %+v", got)
	}

	// The caller's template is never modified.
	_, ts, _ := metricsOf(tpl)
	if ts.(*otlpmetrics.NumberDataPoint).StartTimeUnixNano != 0 {
		t.Error("template was modified")
	}
}

func TestInjectMetricTimestampsKeepsPinnedStartTime(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0)
	e := NewMetricEvolver(templateRequest(), t0)
	r := e.Next(t0.Add(10 * time.Second))

	NewTimestampInjector(0, 0).InjectMetricTimestamps(r.ResourceMetrics[0].ScopeMetrics[0].Metrics[1])
	_, s, _ := metricsOf(r)
	dp := s.(*otlpmetrics.NumberDataPoint)
	if dp.StartTimeUnixNano != uint64(t0.UnixNano()) {
		t.Errorf("start time overwritten: %d", dp.StartTimeUnixNano)
	}
	if dp.TimeUnixNano <= dp.StartTimeUnixNano {
		t.Errorf("data point time %d not after start %d", dp.TimeUnixNano, dp.StartTimeUnixNano)
	}

	// Without a pinned start (static replay), start still defaults to now.
	plain := templateRequest()
	NewTimestampInjector(0, 0).InjectMetricTimestamps(plain.ResourceMetrics[0].ScopeMetrics[0].Metrics[1])
	_, ps, _ := metricsOf(plain)
	if d := ps.(*otlpmetrics.NumberDataPoint); d.StartTimeUnixNano == 0 || d.StartTimeUnixNano != d.TimeUnixNano {
		t.Errorf("static replay start time = %d, time = %d", d.StartTimeUnixNano, d.TimeUnixNano)
	}
}
