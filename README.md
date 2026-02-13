# Telemetry Generator and Sender

A high-performance system for generating and sending large volumes of OpenTelemetry traces, metrics, and logs to test OTLP endpoints at scale.

## Overview

This project consists of two main tools:

1. **telemetry-generator** - Generates template telemetry data (traces, metrics, logs) and saves it to disk
2. **telemetry-sender** - Loads templates and sends them to an OTLP endpoint with high throughput *(coming soon)*

The separation allows you to generate telemetry once and replay it multiple times, making it efficient to test with high event volumes (target: 15M events/second).

## Features

### Generator
- ✅ **Traces**: Generates realistic distributed traces with:
  - Configurable service topology
  - Normal distribution for span counts
  - OpenTelemetry semantic conventions (HTTP, Database)
  - Custom attributes
  - Support for high-span-count traces (100k+ spans)
  
- ✅ **Metrics**: Generates OpenTelemetry metrics:
  - Host metrics (CPU, memory, disk, network)
  - Kubernetes metrics (cluster, node, pod, container)
  - Configurable time series per metric (100-500)
  - Multiple metric types (gauge, sum, histogram)
  
- ✅ **Logs**: Generates structured logs:
  - HTTP access logs (40%)
  - Application logs with severity levels (40%)
  - System logs (20%)

### Sender
- ✅ Load generated templates from protobuf files
- ✅ Add current timestamps with jitter
- ✅ Regenerate trace/span IDs for uniqueness
- ✅ Send to OTLP endpoints via gRPC
- ✅ Configurable rate limiting (events/second)
- ✅ Configurable batch sizes
- ✅ Concurrent workers for high throughput
- ✅ Duration limits and multiplier support
- ✅ Real-time statistics reporting

## Installation

### Option 1: Download Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/honeycomb/telemetry-gen-and-send/releases).

Extract the archive and add the binaries to your PATH.

### Option 2: Docker

Pull the Docker image from GitHub Container Registry:

```bash
# Latest version
docker pull ghcr.io/honeycomb/telemetry-gen-and-send:latest

# Specific version
docker pull ghcr.io/honeycomb/telemetry-gen-and-send:0.1.0
```

Run the tools using Docker:

```bash
# Generate telemetry
docker run -v $(pwd)/generated:/data/generated \
  -v $(pwd)/examples:/config \
  ghcr.io/honeycomb/telemetry-gen-and-send:latest \
  telemetry-generator --config /config/generator-config.yaml --output-dir /data/generated

# Send telemetry
docker run -v $(pwd)/generated:/data/generated \
  -v $(pwd)/examples:/config \
  -e HONEYCOMB_API_KEY="${HONEYCOMB_API_KEY}" \
  ghcr.io/honeycomb/telemetry-gen-and-send:latest \
  telemetry-sender --config /config/sender-config.yaml
```

### Option 3: Build from Source

**Prerequisites:**
- Go 1.23+
- Make

```bash
# Clone the repository
git clone https://github.com/honeycomb/telemetry-gen-and-send.git
cd telemetry-gen-and-send

# Build both tools
make build

# Build just the generator
make generator

# Build just the sender
make sender
```

Binaries will be created in `./build/`

## Usage

### Generating Telemetry

1. Create a configuration file (see `examples/generator-config.yaml`):

```yaml
output:
  directory: "./generated"
  prefix: "telemetry"

traces:
  count: 10000
  spans:
    avg_per_trace: 15
    std_dev: 5
  services:
    count: 5
    names: ["api-gateway", "user-service", "order-service", "payment-service", "inventory-service"]

metrics:
  metric_count: 2000
  timeseries_per_metric:
    min: 100
    max: 500
    default: 300

logs:
  count: 50000
```

2. Run the generator:

```bash
./build/telemetry-generator --config examples/generator-config.yaml

# Optional: Generate JSON output for debugging
./build/telemetry-generator --config examples/generator-config.yaml --json

# Optional: Override output directory
./build/telemetry-generator --config examples/generator-config.yaml --output-dir /tmp/telemetry
```

3. Generated files:
   - `telemetry-traces.pb` - Protobuf binary with trace templates
   - `telemetry-metrics.pb` - Protobuf binary with metric templates
   - `telemetry-logs.pb` - Protobuf binary with log templates
   - `telemetry-metadata.yaml` - Generation metadata

### Sending Telemetry

1. Set your Honeycomb API key:

```bash
export HONEYCOMB_API_KEY="your-api-key-here"
```

2. Review and customize the sender configuration (`examples/sender-config.yaml`):

```yaml
otlp:
  endpoint: "api.honeycomb.io:443"
  headers:
    x-honeycomb-team: "${HONEYCOMB_API_KEY}"

sending:
  rate_limit:
    events_per_second: 10000
  concurrency: 10
  duration: "5m"
  multiplier: 10
```

3. Run the sender:

```bash
./build/telemetry-sender --config examples/sender-config.yaml
```

The sender will:
- Load generated telemetry templates
- Add current timestamps
- Regenerate trace/span IDs
- Send to the OTLP endpoint
- Display real-time statistics every 5 seconds

## Configuration Reference

### Generator Configuration

#### Output
- `output.directory` - Where to write generated files
- `output.prefix` - Prefix for output filenames

#### Traces
- `traces.count` - Number of trace templates
- `traces.spans.avg_per_trace` - Average spans per trace
- `traces.spans.std_dev` - Standard deviation for span counts
- `traces.spans.high_span_traces.enabled` - Generate high-span traces
- `traces.spans.high_span_traces.count` - How many high-span traces
- `traces.spans.high_span_traces.span_count` - Spans in high-span traces
- `traces.services.count` - Number of services
- `traces.services.names` - Service names (optional)
- `traces.services.ingress.single` - Single or multiple ingress services
- `traces.services.ingress.service` - Ingress service name
- `traces.custom_attributes.count` - Number of custom attributes

#### Metrics
- `metrics.metric_count` - Number of distinct metrics
- `metrics.timeseries_per_metric.min` - Minimum time series per metric
- `metrics.timeseries_per_metric.max` - Maximum time series per metric
- `metrics.timeseries_per_metric.default` - Default time series per metric
- `metrics.types` - Metric types to generate (`host_metrics`, `k8s_cluster`, `k8s_node`, `k8s_pod`, `k8s_container`)

#### Logs
- `logs.count` - Number of log templates
- `logs.types.http_access.percentage` - Percentage of HTTP access logs
- `logs.types.application.percentage` - Percentage of application logs
- `logs.types.application.services` - Number of application services
- `logs.types.system.percentage` - Percentage of system logs

### Sender Configuration

#### Input
- `input.traces` - Path to traces protobuf file
- `input.metrics` - Path to metrics protobuf file
- `input.logs` - Path to logs protobuf file

#### OTLP
- `otlp.endpoint` - OTLP gRPC endpoint
- `otlp.headers` - Headers to include (supports `${ENV_VAR}` substitution)
- `otlp.insecure` - Use insecure connection (for localhost testing)

#### Sending
- `sending.rate_limit.events_per_second` - Target throughput (rate limiter controls actual rate)
- `sending.batch_size.traces` - Traces per batch
- `sending.batch_size.metrics` - Metric data points per batch
- `sending.batch_size.logs` - Log records per batch
- `sending.concurrency` - Number of parallel worker goroutines for sending
- `sending.duration` - Maximum time to send ("5m", "1h", "0" for no limit)
- `sending.multiplier` - How many times to replay templates (0 for infinite)

**Note on duration vs multiplier**: The sender stops when **whichever comes first**:
- All multiplier iterations complete, OR
- The duration time expires

Examples:
- `duration: "5m"`, `multiplier: 10` → Stops after 10 replays OR 5 minutes
- `duration: "5m"`, `multiplier: 0` → Runs for exactly 5 minutes (infinite replay)
- `duration: "0"`, `multiplier: 10` → Runs until 10 replays complete (no time limit)
- `duration: "0"`, `multiplier: 0` → Runs indefinitely until manually stopped (Ctrl+C)

#### Timestamps
- `timestamps.jitter_ms` - Random jitter in milliseconds
- `timestamps.backdate_ms` - Backdate timestamps for historical data

## Performance

### Generator
- Generates 10,000 traces in ~300ms
- Generates 2,000 metrics (600k time series) in ~500ms
- Generates 50,000 logs in ~200ms
- Total: ~2MB of telemetry data in under 1 second

### Sender
- Configurable throughput with rate limiting
- Target: 1M+ events/second per instance with tuning
- Concurrent workers for high throughput
- Real-time statistics reporting
- Resource efficient design

## Development

### Project Structure

```
telemetry-gen-and-send/
├── cmd/
│   ├── telemetry-generator/    # Generator CLI
│   └── telemetry-sender/        # Sender CLI (coming soon)
├── internal/
│   ├── config/                  # Configuration parsing
│   ├── generator/               # Generator logic
│   │   ├── common/              # Shared utilities
│   │   ├── traces/              # Trace generation
│   │   ├── metrics/             # Metrics generation
│   │   └── logs/                # Log generation
│   └── sender/                  # Sender logic (coming soon)
├── examples/                    # Example configurations
├── .agents/                     # Development guidelines
├── go.mod
├── Makefile
└── README.md
```

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

### Cleaning Build Artifacts

```bash
make clean
```

## Troubleshooting

### Generator Issues

**Problem**: Out of memory when generating large datasets
**Solution**: Reduce `traces.count`, `metrics.metric_count`, or `logs.count` in your config

**Problem**: Generated files are too large
**Solution**: Reduce time series per metric or number of traces

### Sender Issues

**Problem**: Connection refused or authentication errors
**Solution**: Check your OTLP endpoint and API key in `sender-config.yaml`

**Problem**: Rate too slow or too fast
**Solution**: Adjust `sending.rate_limit.events_per_second` and `sending.concurrency` in config

**Problem**: Sender consumes too much memory
**Solution**: Reduce `sending.multiplier` or generate smaller datasets

## Examples

### Small Test Dataset
```yaml
traces:
  count: 100
metrics:
  metric_count: 50
logs:
  count: 1000
```

### Medium Dataset (Development)
```yaml
traces:
  count: 10000
metrics:
  metric_count: 500
logs:
  count: 50000
```

### Large Dataset (Load Testing)
```yaml
traces:
  count: 100000
metrics:
  metric_count: 2000
logs:
  count: 500000
```

## Contributing

This tool follows the guidelines in `.agents/go.md`. Key points:
- Use Go 1.23+ features
- Follow OpenTelemetry semantic conventions
- Write tests for new functionality
- Keep performance in mind (we're targeting 15M events/sec)

## Releasing

To create a new release:

1. Update the `VERSION` file with the new version number (e.g., `0.2.0`)
2. Commit the version change:
   ```bash
   git add VERSION
   git commit -m "Bump version to 0.2.0"
   ```
3. Create and push a git tag:
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```
4. GitHub Actions will automatically:
   - Build binaries for all platforms (Linux, macOS, Windows)
   - Create a GitHub release with release notes
   - Build and push Docker images to `ghcr.io/honeycomb/telemetry-gen-and-send`
   - Tag the Docker image with the version and `latest`

The release workflow builds for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## License

*(Add your license here)*

## Status

- ✅ Generator: Complete and functional
- ✅ Sender: Complete and functional
- ✅ Docker container: Available on GHCR
- ✅ Multi-platform releases: Automated via GitHub Actions
- 📋 Future enhancements: Performance benchmarks, additional metric types, batch optimization
