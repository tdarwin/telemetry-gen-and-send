# Memory Limits and Performance Guidelines

This document explains memory usage, limits, and best practices for generating and sending high-volume telemetry.

## Overview

The sender loads all generated telemetry templates into memory before sending them. To ensure reliable operation, we enforce a **10GB memory limit** for the sender process.

## Memory Estimation

### Per-Object Memory Usage (Approximate)

| Telemetry Type | Memory per Object |
|----------------|-------------------|
| **Trace Span** | ~2KB |
| **Metric Data Point** | ~400 bytes |
| **Log Record** | ~800 bytes |

### Calculation Formula

```
Total Memory = (Total Spans × 2KB) + (Total Data Points × 400 bytes) + (Total Logs × 800 bytes)
```

**Example:**
- 1,000 traces × 30 spans avg = 30,000 spans → **60 MB**
- 250 metrics × 300 time series = 75,000 data points → **30 MB**
- 10,000 log records → **8 MB**
- **Total: ~98 MB** (well under limit)

## Generator Configuration Limits

### Maximum Values

The generator automatically validates that your configuration won't exceed the 10GB memory limit. If it does, you'll see an error like:

```
Error: estimated sender memory usage (12.34 GB) exceeds maximum (10 GB).
Reduce trace count, spans per trace, high-span traces, metrics, or logs.
```

### Recommended Limits

For optimal performance with the default **10GB limit**:

| Configuration | Recommended Max | Notes |
|---------------|-----------------|-------|
| **Total Spans** | 5,000,000 | Includes normal + high-span traces |
| **Spans per Trace (avg)** | 50 | Higher values require fewer traces |
| **High-Span Traces** | 10,000 spans | Traces >10k spans will be skipped by sender |
| **Metric Data Points** | 25,000,000 | metric_count × timeseries_per_metric |
| **Log Records** | 12,500,000 | |

### High-Span Trace Handling

**Automatic Splitting**: The sender automatically splits traces with >10,000 spans across multiple batches to prevent gRPC message size errors. Each batch contains up to 10,000 spans with the same `trace_id`, and OTLP receivers (like Honeycomb) reassemble them.

**No Limits**: You can safely generate traces with 100,000+ spans:

```yaml
traces:
  spans:
    high_span_traces:
      enabled: true
      count: 10
      span_count: 100000  # Automatically split into 10 batches
```

The sender will split these large traces transparently - no warnings, no data loss.

## Sender Batching Behavior

The sender uses intelligent batching to handle large datasets efficiently:

### Batch Size Configuration

```yaml
sending:
  batch_size:
    traces: 100     # Number of traces per batch
    metrics: 1000   # Number of data points per batch
    logs: 1000      # Number of log records per batch
```

### Span-Count-Aware Batching

The sender automatically:
1. **Limits batches to 10,000 spans** regardless of trace count
2. **Splits large traces across multiple batches** (no data loss)
3. **Dynamically adjusts batch sizes** based on span counts

**Example**: If `batch_size.traces = 100` but the first 10 traces contain 12,000 spans total, the sender will split them into multiple batches to stay under 10k spans per batch.

**Large Trace Example**: A trace with 100,000 spans is automatically split into 10 batches of 10,000 spans each. All batches have the same `trace_id`, and Honeycomb reassembles them into a single trace.

## Memory Usage Monitoring

### Generator Output

The generator shows estimated sender memory:

```
═══════════════════════════════════════════════════════════
  Telemetry Generator
═══════════════════════════════════════════════════════════
Configuration: examples/generator-config.yaml
Output directory: ./generated
Output prefix: telemetry
JSON output: false
Estimated sender memory: 9.87 GB
  ⚠️  High memory usage - consider reducing dataset size
```

### Sender Output

The sender shows batch information:

```
═══════════════════════════════════════════════════════════
Starting 30 worker(s)...
  Trace batches: 105 (batch size: 10 resource spans)
Sending telemetry...
```

Traces are automatically split (no warnings needed):
```
Sending telemetry...
[Stats show all spans sent, including large traces]
```

## Performance Tips

### For Maximum Throughput

1. **Stay under 8GB** to leave room for runtime overhead
2. **Use normal-sized traces** (10-100 spans) rather than high-span traces
3. **Scale horizontally** with multiple sender instances rather than massive single datasets
4. **Increase concurrency** (`sending.concurrency: 50`) if CPU/network allows

### For Large-Scale Testing

**Instead of**:
```yaml
traces:
  count: 100
  spans:
    high_span_traces:
      count: 50
      span_count: 100000  # 5M spans, traces will be skipped!
```

**Use**:
```yaml
traces:
  count: 50000  # More traces
  spans:
    avg_per_trace: 100  # Normal size
  # Result: 5M spans, all sent successfully
```

### Memory-Efficient Patterns

**Pattern 1: Wide Dataset (Many Traces)**
```yaml
traces:
  count: 100000
  spans:
    avg_per_trace: 25
# Result: 2.5M spans = ~5GB
```

**Pattern 2: Deep Dataset (Complex Traces)**
```yaml
traces:
  count: 1000
  spans:
    avg_per_trace: 2500
# Result: 2.5M spans = ~5GB (same total, different shape)
```

Both patterns use similar memory but have different characteristics for testing scenarios.

## Troubleshooting

### "Estimated memory exceeds maximum" Error

**Cause**: Configuration would generate a dataset >10GB.

**Solutions**:
1. Reduce `traces.count`
2. Reduce `traces.spans.avg_per_trace`
3. Reduce or disable `high_span_traces`
4. Reduce `metrics.metric_count` or `timeseries_per_metric`
5. Reduce `logs.count`

### Large Traces Taking Time to Send

**Cause**: Traces with 100k+ spans take time to split and send across multiple batches.

**Expected Behavior**:
- A 100k-span trace is split into 10 batches
- Each batch is cloned, transformed, and sent separately
- This is normal and ensures reliable delivery

**Not a Problem**: The sender handles this automatically.

### Out of Memory (OOM) Errors

**Cause**: System ran out of memory during sender execution.

**Solutions**:
1. Reduce dataset size (see "Estimated memory exceeds maximum")
2. Ensure system has enough RAM (10GB+ recommended)
3. Close other memory-intensive applications
4. Check system swap/virtual memory settings

### Slow Sending Performance

**Cause**: Large batches or high span counts slow down cloning/processing.

**Solutions**:
1. Reduce `sending.batch_size.traces` (try 50 or 100)
2. Increase `sending.concurrency` for more parallelism
3. Check network bandwidth to OTLP endpoint
4. Monitor CPU usage (workers are CPU-bound during cloning)

## Example Configurations

### Small Test (< 100MB)
```yaml
traces:
  count: 100
  spans:
    avg_per_trace: 15
metrics:
  metric_count: 50
logs:
  count: 1000
# Estimated: ~100 MB
```

### Medium Load Test (~ 1GB)
```yaml
traces:
  count: 10000
  spans:
    avg_per_trace: 50
metrics:
  metric_count: 500
logs:
  count: 50000
# Estimated: ~1 GB
```

### Large Load Test (~ 8GB)
```yaml
traces:
  count: 100000
  spans:
    avg_per_trace: 40
    high_span_traces:
      enabled: true
      count: 10
      span_count: 10000
metrics:
  metric_count: 2000
  timeseries_per_metric:
    default: 500
logs:
  count: 500000
# Estimated: ~8 GB (safe maximum)
```

### Maximum Safe Dataset (~ 10GB limit)
```yaml
traces:
  count: 125000
  spans:
    avg_per_trace: 40
metrics:
  metric_count: 2500
  timeseries_per_metric:
    default: 600
logs:
  count: 625000
# Estimated: ~10 GB (at limit)
```

## Technical Details

### Why 10GB?

The 10GB limit ensures:
1. **Reliable operation** on machines with 16GB+ RAM
2. **Room for OS and other processes** (~6GB remaining)
3. **Safe memory headroom** for Go runtime, GC, and buffers
4. **Predictable performance** without thrashing/swapping

### gRPC Message Size Limit

The default gRPC message size limit is **15MB**. At ~200 bytes per span (protobuf wire format), this allows ~75k spans per message. We use a more conservative **10k spans per batch** limit to:
- Ensure reliable delivery
- Reduce memory pressure during cloning
- Allow headroom for large attributes/events

### Memory vs Disk

**Protobuf files on disk** are typically 40-60% the size of in-memory structures due to compression. Example:
- 5M spans in memory: ~10 GB
- 5M spans on disk: ~1.3 GB protobuf file

The sender loads everything into memory for fast replay with `multiplier: 0` (infinite iterations).

## Future Improvements

Potential enhancements for even larger datasets:
1. **Streaming loader**: Load templates incrementally rather than all at once
2. **Chunked high-span traces**: Split very large traces into multiple exports
3. **Configurable memory limit**: Allow users to adjust the 10GB limit
4. **Automatic batch sizing**: Dynamically calculate optimal batch sizes

## Summary

- **10GB memory limit** enforced by generator validation
- **10k spans per batch** limit enforced by sender
- **Use many normal traces** instead of few high-span traces
- **Monitor "Estimated sender memory"** output from generator
- **Scale horizontally** for datasets exceeding limits
