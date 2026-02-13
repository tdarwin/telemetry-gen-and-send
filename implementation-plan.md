# Telemetry Generation and Sending System - Implementation Plan

## Context

This is a greenfield project to build a high-volume telemetry generation and load testing system for Honeycomb. The goal is to generate realistic OpenTelemetry traces, logs, and metrics that can be sent repeatedly to an OTLP endpoint to test scalability up to 15 million events per second.

**Why this is needed:**
- Test Honeycomb's ability to handle extreme ingestion loads
- Create repeatable, consistent telemetry patterns for benchmarking
- Separate generation from sending to allow template reuse and resource efficiency

**Key Design Constraints:**
- Generated telemetry must be timestamp-agnostic (sender adds timestamps)
- Trace/span IDs must be regenerated on each send to avoid duplicates
- Resource efficiency is critical to achieve 15M events/sec target
- Generated data must follow OpenTelemetry semantic conventions

## Architecture Overview

### Two Primary Components

1. **Generator Tool** (`cmd/telemetry-generator`)
   - Generates template telemetry data without timestamps
   - Saves to disk in efficient format
   - Produces traces, metrics, and logs with configurable characteristics

2. **Sender Tool** (`cmd/telemetry-sender`)
   - Loads generated templates from disk
   - Adds current timestamps and regenerates IDs
   - Sends to OTLP endpoint at high volume
   - Supports multiplying templates for scale

## Questions for Discussion

Before finalizing the implementation plan, I need clarification on several technical decisions:

### ✅ 1. Go Version and Dependencies
**DECIDED:** Go 1.25 with official OpenTelemetry Go SDK (`go.opentelemetry.io/*`)
- Easier semantic conventions implementation
- Better maintainability
- Focus on functionality over minimal dependencies

### ✅ 2. Storage Format for Generated Telemetry
**DECIDED:** Dual output format
- **Protobuf binary** - Default output, what sender ingests (efficient)
- **JSON** - Optional debug output via flag (human-readable)
- Generator will support `--json` or `--debug` flag to also output JSON alongside protobuf

### ✅ 3. Configuration Approach
**DECIDED:** YAML config files with CLI path specification
- Configuration in YAML format for complex scenarios
- CLI flags: `--config <path>` to specify config file location
- No hardcoded config paths - always user-specified
- Both generator and sender use YAML configs

### ✅ 4. Metrics Time Series Strategy
**DECIDED:** ~100-500 time series per metric
- Pre-generate dimension combinations
- Configurable in YAML config
- Target middle range (~300) as default

### ✅ 5. Trace Span Distribution
**DECIDED:** Normal distribution
- Spans per trace follow normal distribution around configured average
- Allows natural variance while staying near target

### ✅ 6. Project Structure
**DECIDED:** Approved structure:
```
telemetry-gen-and-send/
├── cmd/
│   ├── telemetry-generator/    # Generator CLI
│   └── telemetry-sender/        # Sender CLI
├── internal/
│   ├── generator/
│   │   ├── traces/
│   │   ├── metrics/
│   │   └── logs/
│   ├── sender/
│   │   ├── loader/
│   │   ├── transformer/
│   │   └── exporter/
│   └── otlp/                    # Shared OTLP types/utils
├── .agents/
│   └── go.md                    # Go dev guidelines
├── go.mod
└── Makefile
```

## Detailed Technical Approach

### Generator Tool (`telemetry-generator`)

#### CLI Interface
```bash
telemetry-generator --config generator-config.yaml [--json] [--output-dir ./output]
```
- `--config`: Path to YAML configuration file (required)
- `--json`: Optional flag to output JSON alongside protobuf for debugging
- `--output-dir`: Directory for output files (default: current directory)

#### YAML Configuration Structure
```yaml
output:
  directory: "./generated"
  prefix: "telemetry"  # Files: telemetry-traces.pb, telemetry-metrics.pb, telemetry-logs.pb

traces:
  count: 10000  # Number of trace templates to generate
  spans:
    avg_per_trace: 15
    std_dev: 5  # For normal distribution
    high_span_traces:
      enabled: true
      count: 10  # Number of high-span traces
      span_count: 100000
  services:
    count: 5
    names: ["api-gateway", "user-service", "order-service", "payment-service", "inventory-service"]
    ingress:
      single: true  # true = one ingress service, false = multiple
      service: "api-gateway"  # Which service is ingress (if single=true)
  custom_attributes:
    count: 10  # Number of custom attributes to randomly add
    # Generator creates attributes like: custom.attr.1 (string), custom.attr.2 (int), etc.

metrics:
  metric_count: 2000  # Number of distinct metric names
  timeseries_per_metric:
    min: 100
    max: 500
    default: 300
  types:
    - host_metrics  # CPU, memory, disk, network
    - k8s_cluster
    - k8s_node
    - k8s_pod
    - k8s_container

logs:
  count: 50000  # Number of log templates
  types:
    http_access:
      percentage: 40  # 40% of logs
    application:
      percentage: 40
      services: 5  # Number of application services
    system:
      percentage: 20
```

#### Trace Generation Logic
1. **Service Topology Creation**
   - Create service graph based on config
   - Define HTTP and database call patterns per service
   - Use OTel semantic conventions: `http.method`, `http.status_code`, `db.system`, `db.statement`

2. **Span Generation (Normal Distribution)**
   - Use Go's `math/rand` with normal distribution (`NormFloat64()`)
   - Formula: `spanCount = max(1, int(avgSpans + stdDev*rand.NormFloat64()))`
   - Generate span relationships (parent-child) forming valid trace tree
   - Add durations without timestamps (relative timing preserved)

3. **Custom Attributes**
   - Pre-define attribute names and types at start
   - Randomly attach to spans (configurable probability)
   - Types: string, int64, float64, bool
   - Example: `custom.request.priority` (int), `custom.user.tier` (string)

4. **High Span Count Traces**
   - Generate separately with specified span counts
   - Maintain valid tree structure (may require multiple levels of depth)

5. **Output Format**
   - Use `go.opentelemetry.io/proto/otlp` protobuf definitions
   - Save as `ExportTraceServiceRequest` messages without timestamps
   - Each trace template includes: span structure, attributes, durations, service info

#### Metrics Generation Logic
1. **Metric Name Generation**
   - Use OTel semantic conventions as base
   - Host metrics: `system.cpu.utilization`, `system.memory.usage`, `system.disk.io`
   - K8s metrics: `k8s.pod.cpu.usage`, `k8s.container.memory.limit`, `k8s.node.network.io`

2. **Time Series Creation**
   - Pre-generate dimension combinations for each metric
   - Target 100-500 time series per metric (configurable)
   - Dimensions examples:
     - Host: `host.name`, `os.type`, `region`, `availability_zone`
     - K8s: `k8s.cluster.name`, `k8s.namespace.name`, `k8s.pod.name`, `container.name`
   - Use realistic dimension values (e.g., generated hostnames, pod names)

3. **Data Point Structure**
   - Generate value templates (no timestamps)
   - Include gauge, counter, and histogram metric types
   - Store dimension sets with metric definitions

4. **Output Format**
   - Save as `ExportMetricsServiceRequest` protobuf messages
   - Each template includes metric descriptors and dimension combinations

#### Log Generation Logic
1. **HTTP Access Logs (40%)**
   - Format: Apache Common Log format translated to OTLP
   - Fields: `http.method`, `http.target`, `http.status_code`, `http.response_size`
   - Generate realistic paths and status code distributions

2. **Application Logs (40%)**
   - Multiple service names (configurable count)
   - Severity levels: DEBUG, INFO, WARN, ERROR (weighted distribution)
   - Structured fields: `service.name`, `level`, `message`, `error.type`
   - Realistic log messages per severity

3. **System Logs (20%)**
   - System-level events: startup, shutdown, resource alerts
   - Fields: `log.source`, `event.type`, `severity`

4. **Output Format**
   - Save as `ExportLogsServiceRequest` protobuf messages
   - No timestamps in body fields

#### File Output Structure
```
<output-dir>/
├── telemetry-traces.pb      # Protobuf binary (primary)
├── telemetry-traces.json     # JSON (if --json flag used)
├── telemetry-metrics.pb
├── telemetry-metrics.json    # (optional)
├── telemetry-logs.pb
├── telemetry-logs.json       # (optional)
└── metadata.yaml             # Generation metadata: config used, counts, timestamp
```

### Sender Tool (`telemetry-sender`)

#### CLI Interface
```bash
telemetry-sender --config sender-config.yaml
```

#### YAML Configuration Structure
```yaml
input:
  traces: "./generated/telemetry-traces.pb"
  metrics: "./generated/telemetry-metrics.pb"
  logs: "./generated/telemetry-logs.pb"

otlp:
  endpoint: "api.honeycomb.io:443"
  headers:
    x-honeycomb-team: "${HONEYCOMB_API_KEY}"
  insecure: false

sending:
  rate_limit:
    events_per_second: 1000000  # Target throughput
  batch_size:
    traces: 100    # Traces per batch
    metrics: 1000  # Metric data points per batch
    logs: 1000     # Log records per batch
  concurrency: 50  # Number of concurrent goroutines
  duration: "10m"  # How long to send (0 = infinite)
  multiplier: 10   # How many times to reuse templates

timestamps:
  jitter_ms: 1000  # Add random jitter up to 1s to timestamps
  backdate_ms: 0   # How far back to start timestamps (for historical data)
```

#### Implementation Details

1. **Template Loading**
   - Load protobuf files into memory using `proto.Unmarshal`
   - Parse into OTel protobuf structures
   - Store in efficient in-memory representation
   - Pre-allocate buffers for transformations

2. **Timestamp Injection**
   - Calculate current time: `time.Now()`
   - Add configurable jitter: `time.Now().Add(-time.Duration(rand.Intn(jitterMs)) * time.Millisecond)`
   - For traces: maintain relative span timings
     - Root span: current timestamp - total trace duration
     - Child spans: root timestamp + relative offsets
   - For metrics: current timestamp
   - For logs: current timestamp with jitter

3. **ID Regeneration (Traces)**
   - Generate new `TraceID` for each trace: 16 random bytes
   - Generate new `SpanID` for each span: 8 random bytes
   - Preserve parent-child relationships using new IDs
   - Use `crypto/rand` for uniqueness

4. **Metrics Time Series Multiplication**
   - Take existing dimension combinations
   - Add additional synthetic dimensions if needed
   - Generate extra time series by varying dimension values
   - Maintain realistic cardinality

5. **Batching and Export**
   - Group events into batches per config
   - Use `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
   - Use `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`
   - Use `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc`
   - Concurrent workers (goroutines) send batches in parallel

6. **Rate Limiting**
   - Use `golang.org/x/time/rate` for token bucket rate limiting
   - Distribute rate across concurrent workers
   - Monitor actual throughput vs target

7. **Error Handling and Retries**
   - Retry failed batches with exponential backoff
   - Log errors without stopping send process
   - Track success/failure metrics

8. **Progress Monitoring**
   - Print periodic stats: events sent, events/sec, errors
   - Track per-signal type (traces, metrics, logs)
   - Estimate completion time

### Performance Considerations

1. **Memory Efficiency**
   - Stream processing where possible
   - Reuse buffers and objects (sync.Pool)
   - Limit in-memory template copies

2. **CPU Efficiency**
   - Minimize allocations in hot path
   - Use efficient protobuf marshaling
   - Parallel processing with controlled concurrency

3. **Network Efficiency**
   - HTTP/2 connection pooling
   - Batch size optimization
   - Compression (gzip) for OTLP requests

4. **Scalability**
   - Multiple sender instances can load same templates
   - Each sender generates unique IDs
   - No coordination needed between senders

## Implementation Steps

### Phase 1: Project Setup
1. **Initialize Go Module**
   - Create `go.mod` with Go 1.25
   - Add dependencies:
     - `go.opentelemetry.io/otel`
     - `go.opentelemetry.io/proto/otlp`
     - `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
     - `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`
     - `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc`
     - `gopkg.in/yaml.v3` (for config parsing)
     - `google.golang.org/protobuf`
     - `golang.org/x/time/rate` (for rate limiting)

2. **Create Directory Structure**
   - Set up `cmd/`, `internal/` directories
   - Create subdirectories per architecture diagram
   - Add `.agents/go.md` with Go development guidelines

3. **Create Makefile**
   - Build targets for generator and sender
   - Test targets
   - Clean targets
   - Install targets

### Phase 2: Generator - Configuration & Foundation
1. **Config Package** (`internal/config/generator.go`)
   - Define structs matching YAML structure
   - YAML parsing with validation
   - Default values

2. **CLI Entry Point** (`cmd/telemetry-generator/main.go`)
   - Flag parsing (`--config`, `--json`, `--output-dir`)
   - Config loading
   - Orchestrate generation flow
   - Error handling and logging

3. **Common Utilities** (`internal/generator/common/`)
   - Random value generators (strings, numbers, booleans)
   - Name generators (service names, hostnames, pod names)
   - OTel attribute helpers
   - Protobuf serialization helpers

### Phase 3: Generator - Trace Generation
1. **Service Topology** (`internal/generator/traces/topology.go`)
   - Build service graph from config
   - Define service call patterns
   - HTTP vs Database operation distributions

2. **Span Generator** (`internal/generator/traces/spans.go`)
   - Normal distribution span count logic
   - Span tree builder (parent-child relationships)
   - Duration assignment (realistic timings)
   - OTel semantic attributes (HTTP, DB conventions)

3. **Custom Attributes** (`internal/generator/traces/attributes.go`)
   - Attribute schema definition
   - Random attribute attachment logic

4. **High Span Traces** (`internal/generator/traces/high_span.go`)
   - Special handling for 100k+ span traces
   - Efficient tree generation
   - Memory-conscious approach

5. **Trace Writer** (`internal/generator/traces/writer.go`)
   - Assemble protobuf `ExportTraceServiceRequest`
   - Write protobuf binary
   - Optional JSON output

### Phase 4: Generator - Metrics Generation
1. **Metric Schema** (`internal/generator/metrics/schema.go`)
   - Define OTel host metrics (CPU, memory, disk, network)
   - Define K8s metrics (cluster, node, pod, container)
   - Metric type definitions (gauge, counter, histogram)

2. **Dimension Generator** (`internal/generator/metrics/dimensions.go`)
   - Create dimension combinations
   - Generate 100-500 time series per metric
   - Realistic dimension values

3. **Metrics Writer** (`internal/generator/metrics/writer.go`)
   - Assemble protobuf `ExportMetricsServiceRequest`
   - Write protobuf binary
   - Optional JSON output

### Phase 5: Generator - Log Generation
1. **Log Templates** (`internal/generator/logs/templates.go`)
   - HTTP access log structure
   - Application log messages by severity
   - System log events

2. **Log Generator** (`internal/generator/logs/generator.go`)
   - 40/40/20 distribution logic
   - Severity distribution (weighted)
   - Structured field generation

3. **Logs Writer** (`internal/generator/logs/writer.go`)
   - Assemble protobuf `ExportLogsServiceRequest`
   - Write protobuf binary
   - Optional JSON output

### Phase 6: Sender - Configuration & Loading
1. **Sender Config** (`internal/config/sender.go`)
   - Define sender YAML structure
   - Parse and validate
   - Environment variable substitution for API keys

2. **Template Loader** (`internal/sender/loader/loader.go`)
   - Load protobuf files
   - Unmarshal into OTel structures
   - Efficient in-memory representation

### Phase 7: Sender - Transformation
1. **Timestamp Injector** (`internal/sender/transformer/timestamps.go`)
   - Current time calculation
   - Jitter logic
   - Trace relative timing preservation
   - Backdate support

2. **ID Regenerator** (`internal/sender/transformer/ids.go`)
   - TraceID generation (16 bytes)
   - SpanID generation (8 bytes)
   - Parent-child relationship preservation

3. **Metrics Multiplier** (`internal/sender/transformer/metrics.go`)
   - Time series multiplication logic
   - Additional dimension generation

### Phase 8: Sender - Export & Rate Limiting
1. **OTLP Exporters** (`internal/sender/exporter/`)
   - `traces.go` - gRPC trace exporter setup
   - `metrics.go` - gRPC metrics exporter setup
   - `logs.go` - gRPC logs exporter setup
   - Connection management and pooling

2. **Batcher** (`internal/sender/batch/batcher.go`)
   - Batch assembly per config
   - Separate batching for traces/metrics/logs

3. **Rate Limiter** (`internal/sender/ratelimit/limiter.go`)
   - Token bucket implementation
   - Per-worker rate distribution
   - Throughput monitoring

4. **Worker Pool** (`internal/sender/workers/pool.go`)
   - Concurrent goroutines
   - Work distribution
   - Error handling and retries

### Phase 9: Sender - CLI & Orchestration
1. **CLI Entry Point** (`cmd/telemetry-sender/main.go`)
   - Flag parsing (`--config`)
   - Config loading
   - Initialize exporters
   - Start worker pool
   - Progress monitoring
   - Graceful shutdown

2. **Stats Reporter** (`internal/sender/stats/reporter.go`)
   - Periodic stats output
   - Events/sec calculation
   - Success/failure tracking

### Phase 10: Testing & Documentation
1. **Unit Tests**
   - Test normal distribution logic
   - Test ID generation uniqueness
   - Test timestamp calculations
   - Test config parsing

2. **Integration Tests**
   - End-to-end: generate → send → verify
   - Test with local OTLP receiver

3. **Example Configurations**
   - Create `examples/generator-config.yaml` with sample configuration
   - Create `examples/sender-config.yaml` with sample configuration
   - Include comments explaining each option

4. **README.md**
   - Project overview and purpose
   - Quick start guide
   - Installation instructions
   - Usage examples for both generator and sender
   - Configuration reference
   - Performance tuning guide
   - Troubleshooting section

## Critical Files

### Generator
- `cmd/telemetry-generator/main.go`
- `internal/config/generator.go`
- `internal/generator/traces/generator.go`, `topology.go`, `spans.go`, `writer.go`
- `internal/generator/metrics/schema.go`, `dimensions.go`, `writer.go`
- `internal/generator/logs/templates.go`, `generator.go`, `writer.go`
- `internal/generator/common/random.go`, `names.go`, `attributes.go`

### Sender
- `cmd/telemetry-sender/main.go`
- `internal/config/sender.go`
- `internal/sender/loader/loader.go`
- `internal/sender/transformer/timestamps.go`, `ids.go`, `metrics.go`
- `internal/sender/exporter/traces.go`, `metrics.go`, `logs.go`
- `internal/sender/batch/batcher.go`
- `internal/sender/ratelimit/limiter.go`
- `internal/sender/workers/pool.go`
- `internal/sender/stats/reporter.go`

### Build & Config
- `go.mod`
- `Makefile`
- `.agents/go.md`

## Key Dependencies

```go.mod
module github.com/honeycomb/telemetry-gen-and-send

go 1.25

require (
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/proto/otlp v1.1.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v1.24.0
    google.golang.org/protobuf v1.32.0
    gopkg.in/yaml.v3 v3.0.1
    golang.org/x/time v0.5.0
    google.golang.org/grpc v1.62.0
)
```

## Verification Plan

### Generator Verification
1. **Output Validation**
   ```bash
   # Generate with default config
   ./telemetry-generator --config examples/generator-config.yaml --json
   
   # Verify files created
   ls -lh generated/
   # Should see: telemetry-traces.pb, telemetry-metrics.pb, telemetry-logs.pb
   # And if --json used: *.json versions
   
   # Check metadata
   cat generated/metadata.yaml
   ```

2. **Trace Validation**
   - Inspect JSON output (if generated) to verify:
     - Span counts follow normal distribution around average
     - High span traces have correct count
     - Parent-child relationships are valid (no orphans)
     - OTel semantic conventions present (`http.method`, `db.system`, etc.)
     - Custom attributes appear with consistent types
     - No timestamps in span data
   
3. **Metrics Validation**
   - Verify metric count matches config
   - Check time series per metric in range 100-500
   - Validate dimension combinations are realistic
   - Confirm metric types (gauge, counter, histogram)
   - No timestamps in data points

4. **Logs Validation**
   - Verify log type distribution (40/40/20)
   - Check severity distribution in application logs
   - Validate structured fields present
   - No timestamps in log records

### Sender Verification
1. **Template Loading**
   ```bash
   # Start sender (will load templates and begin sending)
   ./telemetry-sender --config examples/sender-config.yaml
   
   # Should see output like:
   # Loaded 10000 trace templates
   # Loaded 2000 metric templates (600000 time series)
   # Loaded 50000 log templates
   # Starting 50 workers...
   ```

2. **Timestamp Verification**
   - Query Honeycomb for data in "last 10 minutes"
   - Verify events appear with recent timestamps
   - Check that trace span timings are relative (root span older than child spans)

3. **ID Uniqueness Verification**
   - Query multiple traces with same structure
   - Verify TraceIDs and SpanIDs are all unique
   - Confirm no duplicate events

4. **Throughput Verification**
   - Monitor sender stats output
   - Should report events/sec near configured rate limit
   - Check Honeycomb ingestion to match sender reports

### Performance Testing
1. **Single Sender Performance**
   ```bash
   # Test with increasing rate limits
   # Edit sender-config.yaml, set events_per_second to 100k, 500k, 1M
   ./telemetry-sender --config sender-config.yaml
   
   # Monitor CPU and memory
   top -p $(pgrep telemetry-sender)
   ```

2. **Multi-Sender Scaling**
   ```bash
   # Run multiple senders in parallel
   ./telemetry-sender --config sender1.yaml &
   ./telemetry-sender --config sender2.yaml &
   ./telemetry-sender --config sender3.yaml &
   
   # Aggregate throughput should approach 15M events/sec
   ```

3. **Resource Monitoring**
   - Track memory usage (should stay bounded)
   - Monitor CPU utilization
   - Check network bandwidth usage
   - Verify no memory leaks over extended runs

### End-to-End Verification
1. **Complete Flow**
   ```bash
   # Step 1: Generate templates
   ./telemetry-generator --config examples/generator-config.yaml
   
   # Step 2: Send to Honeycomb
   export HONEYCOMB_API_KEY="your-key"
   ./telemetry-sender --config examples/sender-config.yaml
   
   # Step 3: Query in Honeycomb UI
   # - Check "last 10 minutes" shows recent data
   # - Verify traces show correct span structure
   # - Check metrics have expected time series
   # - Validate logs appear with correct fields
   ```

2. **Data Quality Checks in Honeycomb**
   - **Traces**: Verify trace waterfall shows parent-child relationships
   - **Traces**: Check for OTel semantic convention attributes
   - **Metrics**: Query specific metric names, verify time series count
   - **Logs**: Filter by log type, verify distribution
   - **Logs**: Check severity levels present in application logs

3. **Continuous Operation**
   - Run sender for extended period (30+ minutes)
   - Verify consistent throughput
   - Check for any errors or retries in sender logs
   - Confirm data continues to appear in Honeycomb "last 10 minutes"

### Success Criteria
- ✅ Generator produces valid protobuf files
- ✅ JSON debug output (when enabled) is human-readable
- ✅ Traces have correct structure and semantic conventions
- ✅ Metrics follow OTel host/K8s patterns with proper cardinality
- ✅ Logs have correct type distribution and fields
- ✅ Sender loads templates without errors
- ✅ Timestamps are current (within configured jitter)
- ✅ Trace/Span IDs are unique across sends
- ✅ Target throughput achieved (1M+ events/sec per sender)
- ✅ Resource usage is reasonable (memory bounded, CPU < 100% per sender)
- ✅ Data appears correctly in Honeycomb UI
- ✅ Extended runs are stable (no crashes, memory leaks, or degradation)
