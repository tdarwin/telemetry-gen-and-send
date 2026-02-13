# Container Deployment Guide

This document provides a comprehensive guide to deploying the Telemetry Generator and Sender using Docker Compose or Kubernetes (Helm).

## Table of Contents

- [Overview](#overview)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Kubernetes Helm Deployment](#kubernetes-helm-deployment)
- [Architecture Comparison](#architecture-comparison)
- [Testing Verification](#testing-verification)

## Overview

Both deployment methods provide:
- **Automatic generator execution**: Templates are created before sending starts
- **Continuous operation**: Senders run indefinitely by default (configurable)
- **Horizontal scaling**: Multiple sender instances for higher throughput
- **Shared template storage**: Generator creates once, senders read many times
- **Environment-based configuration**: Easy customization without rebuilding

## Docker Compose Deployment

### Files Created

```
docker-compose.yml                    # Main orchestration file
deploy/configs/generator.yaml         # Container-optimized generator config
deploy/configs/sender.yaml            # Container-optimized sender config
.env.example                          # Environment variable template
```

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose                       │
│                                                          │
│  ┌──────────────┐                                       │
│  │  Generator   │                                       │
│  │  (run once)  │─────┐                                │
│  └──────────────┘     │                                │
│                       ▼                                 │
│                 Named Volume                            │
│                 (telemetry-data)                        │
│                       │                                 │
│  ┌────────────────────┴─────────────────┐              │
│  │    Sender 1      Sender 2    ...     │              │
│  │  (continuous)  (continuous)           │──▶ OTLP     │
│  └──────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────┘
```

### Quick Start

1. **Setup environment**:
   ```bash
   cp .env.example .env
   # Edit .env and set HONEYCOMB_API_KEY
   ```

2. **Start services**:
   ```bash
   # Single sender
   docker compose up

   # Multiple senders (scale horizontally)
   docker compose up --scale sender=10

   # Background mode
   docker compose up -d --scale sender=5
   ```

3. **Monitor**:
   ```bash
   docker compose logs -f sender
   docker stats
   ```

4. **Stop**:
   ```bash
   docker compose down
   ```

### Configuration

All configuration via environment variables in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `HONEYCOMB_API_KEY` | (required) | Honeycomb API key |
| `OTLP_ENDPOINT` | `api.honeycomb.io:443` | OTLP gRPC endpoint |
| `RATE_LIMIT_EPS` | `100000` | Events per second per sender |
| `CONCURRENCY` | `30` | Worker goroutines per sender |
| `DURATION` | `0` | Duration to send (0 = continuous) |
| `MULTIPLIER` | `0` | Template replay count (0 = infinite) |

### Use Cases

**Continuous load testing** (default):
```bash
docker compose up --scale sender=5
```
Runs indefinitely, total throughput: 5 × 100k = 500k events/sec

**Time-limited test**:
```bash
DURATION=5m docker compose up
```
Automatically stops after 5 minutes

**High throughput**:
```bash
RATE_LIMIT_EPS=500000 CONCURRENCY=50 docker compose up --scale sender=20
```
Total throughput: 20 × 500k = 10M events/sec

## Kubernetes Helm Deployment

### Files Created

```
helm/telemetry-gen-and-send/
├── Chart.yaml                         # Chart metadata
├── values.yaml                        # Default configuration
├── README.md                          # Comprehensive documentation
├── .helmignore                        # Files to exclude from package
└── templates/
    ├── _helpers.tpl                   # Template helpers
    ├── deployment.yaml                # Main deployment (init + main container)
    ├── configmap-generator.yaml       # Generator config
    ├── configmap-sender.yaml          # Sender config
    ├── secret.yaml                    # Honeycomb API key
    ├── serviceaccount.yaml            # Service account
    ├── pvc.yaml                       # Optional persistent volume
    └── NOTES.txt                      # Post-install instructions
```

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│                   Kubernetes Cluster                     │
│                                                          │
│  ┌────────────────────────────────────────────────┐     │
│  │           Deployment (N replicas)               │     │
│  │                                                 │     │
│  │  ┌───────────────────────────────────────┐     │     │
│  │  │ Pod 1                                 │     │     │
│  │  │  ┌──────────┐    ┌────────────┐      │     │     │
│  │  │  │  Init:   │    │   Main:    │      │     │     │
│  │  │  │Generator │───▶│  Sender    │──────┼─────┼───▶ OTLP
│  │  │  └──────────┘    └────────────┘      │     │     │
│  │  │       │                 │             │     │     │
│  │  │       └───emptyDir──────┘             │     │     │
│  │  └───────────────────────────────────────┘     │     │
│  │                                                 │     │
│  │  ┌───────────────────────────────────────┐     │     │
│  │  │ Pod N (same structure)                │     │     │
│  │  └───────────────────────────────────────┘     │     │
│  └────────────────────────────────────────────────┘     │
│                                                          │
│  ConfigMaps: generator.yaml, sender.yaml                │
│  Secret: Honeycomb API key                              │
└──────────────────────────────────────────────────────────┘
```

### Quick Start

1. **Install with inline API key**:
   ```bash
   helm install load-test ./helm/telemetry-gen-and-send \
     --set honeycomb.apiKey="your-api-key"
   ```

2. **Or use existing secret** (recommended):
   ```bash
   kubectl create secret generic honeycomb-creds \
     --from-literal=api-key="your-api-key"

   helm install load-test ./helm/telemetry-gen-and-send \
     --set honeycomb.existingSecret="honeycomb-creds"
   ```

3. **Monitor**:
   ```bash
   kubectl get pods -l app.kubernetes.io/name=telemetry-gen-and-send
   kubectl logs -l app.kubernetes.io/name=telemetry-gen-and-send -c sender -f
   ```

4. **Scale**:
   ```bash
   kubectl scale deployment load-test-telemetry-gen-and-send --replicas=20
   ```

5. **Uninstall**:
   ```bash
   helm uninstall load-test
   ```

### Configuration

Key parameters in Helm values:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `replicaCount` | `3` | Number of sender pods |
| `honeycomb.apiKey` | `""` | Honeycomb API key (inline) |
| `honeycomb.existingSecret` | `""` | Use existing secret |
| `sender.config.otlp.endpoint` | `api.honeycomb.io:443` | OTLP endpoint |
| `sender.config.sending.rateLimit.eventsPerSecond` | `100000` | Events/sec per pod |
| `sender.config.sending.concurrency` | `30` | Workers per pod |
| `sender.config.sending.duration` | `"0"` | Duration (0 = continuous) |
| `sender.config.sending.multiplier` | `0` | Replay count (0 = infinite) |

### Use Cases

**Continuous load testing** (default):
```bash
helm install continuous ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="key" \
  --set replicaCount=10
```
Total throughput: 10 pods × 100k = 1M events/sec, runs until uninstalled

**Time-limited test**:
```bash
helm install time-limited ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="key" \
  --set sender.config.sending.duration="10m"
```
Automatically stops after 10 minutes

**High throughput**:
```bash
helm install high-throughput ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="key" \
  --set replicaCount=50 \
  --set sender.config.sending.rateLimit.eventsPerSecond=1000000
```
Total throughput: 50 pods × 1M = 50M events/sec

## Architecture Comparison

| Feature | Docker Compose | Kubernetes Helm |
|---------|----------------|-----------------|
| **Generator Pattern** | Single service, runs once | Init container per pod |
| **Sender Scaling** | `--scale sender=N` | `kubectl scale` or `replicaCount` |
| **Volume** | Named volume (shared) | emptyDir (per-pod) or PVC |
| **Configuration** | Environment variables | ConfigMaps + Secrets |
| **Orchestration** | `depends_on` | Init container pattern |
| **Best For** | Development, testing | Production, high availability |
| **Horizontal Scaling** | Manual scaling | Deployment controller |
| **Resource Limits** | Deploy resources | Native requests/limits |
| **High Availability** | Single host | Multi-node distribution |

## Testing Verification

### Docker Compose Tests

1. **Basic functionality**:
   ```bash
   export HONEYCOMB_API_KEY="test-key"
   docker compose up
   # Verify: generator runs and exits, sender starts
   ```

2. **Scaling**:
   ```bash
   docker compose up --scale sender=5
   # Verify: 5 sender containers running
   ```

3. **Volume persistence**:
   ```bash
   docker compose down
   docker compose up
   # Verify: generator runs again, new templates created
   ```

4. **Continuous operation**:
   ```bash
   docker compose up -d
   docker compose logs -f sender
   # Let run 2-3 minutes, verify continuous sending
   docker compose down
   ```

5. **Time-limited mode**:
   ```bash
   DURATION=30s docker compose up
   # Verify: senders stop after 30 seconds
   ```

### Helm Chart Tests

1. **Lint and validate**:
   ```bash
   helm lint ./helm/telemetry-gen-and-send
   helm template test ./helm/telemetry-gen-and-send --set honeycomb.apiKey="test"
   ```

2. **Install to cluster**:
   ```bash
   kubectl create namespace telemetry-test
   helm install test ./helm/telemetry-gen-and-send \
     -n telemetry-test \
     --set honeycomb.apiKey="test-key" \
     --set replicaCount=3
   ```

3. **Verify init container**:
   ```bash
   kubectl get pods -n telemetry-test
   kubectl logs <pod> -c generator -n telemetry-test
   # Verify: generator completes successfully
   ```

4. **Verify sender operation**:
   ```bash
   kubectl logs <pod> -c sender -n telemetry-test -f
   # Verify: sender reads templates and sends to OTLP
   ```

5. **Scaling test**:
   ```bash
   kubectl scale deployment test-telemetry-gen-and-send --replicas=10 -n telemetry-test
   # Verify: 10 pods start successfully
   ```

6. **ConfigMap updates**:
   ```bash
   helm upgrade test ./helm/telemetry-gen-and-send \
     --set sender.config.sending.rateLimit.eventsPerSecond=200000 \
     -n telemetry-test
   # Verify: pods restart due to checksum change
   ```

7. **Continuous operation**:
   ```bash
   kubectl logs <pod> -c sender -n telemetry-test -f
   # Let run 2-3 minutes, observe continuous sending
   helm uninstall test -n telemetry-test
   # Verify: graceful shutdown
   ```

## Troubleshooting

### Docker Compose

**Problem**: Generator fails to start
```bash
docker compose logs generator
```
Check: Volume permissions, config file path

**Problem**: Sender can't connect to OTLP
```bash
docker compose logs sender
```
Check: `OTLP_ENDPOINT`, `HONEYCOMB_API_KEY`, network connectivity

**Problem**: Low throughput
- Increase `RATE_LIMIT_EPS` and `CONCURRENCY`
- Scale senders: `docker compose up --scale sender=10`

### Helm Chart

**Problem**: Pods stuck in Init
```bash
kubectl describe pod <pod-name>
kubectl logs <pod-name> -c generator
```
Check: Resources, volume mounts, config

**Problem**: Sender not sending
```bash
kubectl logs <pod-name> -c sender
```
Check: Secret (API key), endpoint, network policies

**Problem**: High resource usage
```bash
kubectl top pods
helm upgrade ... --set sender.config.sending.concurrency=20 --reuse-values
```

## Continuous Operation Details

Both deployments default to **continuous operation**:
- `duration: "0"` = no time limit
- `multiplier: 0` = infinite replay

**To run continuously** (default):
- Docker Compose: `docker compose up` (runs until `docker compose down`)
- Helm: `helm install ...` (runs until `helm uninstall`)

**To run time-limited**:
- Docker Compose: `DURATION=5m docker compose up`
- Helm: `--set sender.config.sending.duration="5m"`

**Stopping continuous senders**:
- Docker Compose: `docker compose down` (sends SIGTERM, graceful shutdown)
- Helm: `helm uninstall <release>` (sends SIGTERM to pods)

Senders handle SIGTERM gracefully (finish current batch, then exit).

## Next Steps

For detailed configuration and advanced use cases:
- [Helm Chart README](./helm/telemetry-gen-and-send/README.md)
- [Main README](./README.md)
- [Example Configs](./examples/)
