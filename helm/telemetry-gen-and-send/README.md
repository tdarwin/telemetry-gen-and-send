# Telemetry Gen and Send Helm Chart

High-performance OpenTelemetry trace, metric, and log generator for load testing Honeycomb and other OTLP endpoints.

## Architecture

This Helm chart deploys a Kubernetes Deployment with:

- **Init Container (Generator)**: Runs once per pod to create telemetry templates (protobuf binaries)
- **Main Container (Sender)**: Continuously reads templates and sends to OTLP endpoint
- **Shared Volume**: emptyDir (default) or PVC for template storage

Each pod generates its own templates during initialization, ensuring isolation and independent operation.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Honeycomb API key (or other OTLP endpoint credentials)

## Installation

### Quick Start

```bash
# Install with inline API key
helm install my-load-test ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key-here"

# Or create secret first (recommended for production)
kubectl create secret generic honeycomb-credentials \
  --from-literal=api-key="your-api-key-here"

helm install my-load-test ./helm/telemetry-gen-and-send \
  --set honeycomb.existingSecret="honeycomb-credentials"
```

### Install from Repository (once published)

```bash
helm repo add honeycomb https://honeycomb.io/helm-charts
helm repo update

helm install my-load-test honeycomb/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key-here"
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of sender pods | `3` |
| `honeycomb.apiKey` | Honeycomb API key (inline) | `""` |
| `honeycomb.existingSecret` | Name of existing secret with API key | `""` |
| `sender.config.otlp.endpoint` | OTLP gRPC endpoint | `api.honeycomb.io:443` |
| `sender.config.sending.rateLimit.eventsPerSecond` | Events per second per pod | `100000` |
| `sender.config.sending.concurrency` | Worker goroutines per pod | `30` |
| `sender.config.sending.duration` | Max send duration (0 = continuous) | `"0"` |
| `sender.config.sending.multiplier` | Template replay count (0 = infinite) | `0` |
| `storage.type` | Storage type: `emptyDir` or `persistentVolumeClaim` | `emptyDir` |

### Full values.yaml

See [values.yaml](./values.yaml) for all available configuration options.

## Common Use Cases

### 1. Continuous Load Testing (Default)

Run indefinitely until manually stopped:

```bash
helm install continuous-load ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key" \
  --set replicaCount=5 \
  --set sender.config.sending.rateLimit.eventsPerSecond=200000
```

**Total throughput**: 5 pods × 200k events/sec = **1M events/sec**

To stop:
```bash
helm uninstall continuous-load
```

### 2. Time-Limited Load Test

Run for a specific duration:

```bash
helm install time-limited ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key" \
  --set sender.config.sending.duration="10m" \
  --set replicaCount=10
```

Senders will automatically stop after 10 minutes.

### 3. High-Throughput Test

Scale to maximum throughput:

```bash
helm install high-throughput ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key" \
  --set replicaCount=20 \
  --set sender.config.sending.rateLimit.eventsPerSecond=500000 \
  --set sender.config.sending.concurrency=50 \
  --set sender.resources.limits.cpu="4000m" \
  --set sender.resources.limits.memory="2Gi"
```

**Total throughput**: 20 pods × 500k events/sec = **10M events/sec**

### 4. Custom OTLP Endpoint

Send to non-Honeycomb endpoint:

```bash
helm install custom-endpoint ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-token" \
  --set sender.config.otlp.endpoint="localhost:4317" \
  --set sender.config.otlp.insecure=true
```

### 5. Persistent Template Storage

Use a PVC to persist templates across pod restarts:

```bash
helm install persistent ./helm/telemetry-gen-and-send \
  --set honeycomb.apiKey="your-api-key" \
  --set storage.type="persistentVolumeClaim" \
  --set storage.persistentVolumeClaim.size="10Gi" \
  --set storage.persistentVolumeClaim.storageClass="fast-ssd"
```

Note: With PVC, only the first pod initialization generates templates. Subsequent pods reuse them.

## Monitoring

### Check Deployment Status

```bash
kubectl get deployment -l app.kubernetes.io/name=telemetry-gen-and-send
kubectl get pods -l app.kubernetes.io/name=telemetry-gen-and-send
```

### View Logs

```bash
# Generator logs (init container)
kubectl logs -l app.kubernetes.io/name=telemetry-gen-and-send -c generator

# Sender logs (main container) - follow
kubectl logs -l app.kubernetes.io/name=telemetry-gen-and-send -c sender -f

# Single pod logs
kubectl logs <pod-name> -c sender -f
```

### Check Resource Usage

```bash
kubectl top pods -l app.kubernetes.io/name=telemetry-gen-and-send
```

## Scaling

### Scale Horizontally (More Pods)

```bash
# Via kubectl
kubectl scale deployment <deployment-name> --replicas=10

# Via Helm upgrade
helm upgrade my-load-test ./helm/telemetry-gen-and-send \
  --set replicaCount=10 \
  --reuse-values
```

Each new pod:
- Runs generator init container to create templates
- Starts sender container independently
- Contributes to total throughput

### Scale Vertically (More Resources)

```bash
helm upgrade my-load-test ./helm/telemetry-gen-and-send \
  --set sender.config.sending.concurrency=100 \
  --set sender.config.sending.rateLimit.eventsPerSecond=1000000 \
  --set sender.resources.limits.cpu="8000m" \
  --set sender.resources.limits.memory="4Gi" \
  --reuse-values
```

## Upgrading

### Update Configuration

```bash
helm upgrade my-load-test ./helm/telemetry-gen-and-send \
  --set sender.config.sending.rateLimit.eventsPerSecond=500000 \
  --reuse-values
```

Pods will automatically restart when ConfigMaps change (checksum annotations trigger rolling update).

### Update Image Version

```bash
helm upgrade my-load-test ./helm/telemetry-gen-and-send \
  --set image.tag="v0.2.0" \
  --reuse-values
```

## Uninstalling

```bash
helm uninstall my-load-test
```

This removes:
- Deployment (all pods)
- ConfigMaps
- Secret (if created by chart)
- ServiceAccount

Persistent volumes (if used) may be retained based on reclaim policy.

## Troubleshooting

### Pods Stuck in Init:0/1

Generator init container is running or failing.

```bash
kubectl logs <pod-name> -c generator
kubectl describe pod <pod-name>
```

Common causes:
- Insufficient resources
- Volume mount issues
- Configuration errors

### Sender Not Sending

```bash
kubectl logs <pod-name> -c sender
```

Common causes:
- Invalid API key (check secret)
- Wrong OTLP endpoint
- Network connectivity issues
- Rate limiting by endpoint

### High CPU/Memory Usage

Adjust resources or reduce concurrency:

```bash
helm upgrade my-load-test ./helm/telemetry-gen-and-send \
  --set sender.config.sending.concurrency=20 \
  --set sender.resources.limits.cpu="2000m" \
  --reuse-values
```

### Pods Not Spreading Across Nodes

Check pod anti-affinity:

```bash
kubectl get pods -l app.kubernetes.io/name=telemetry-gen-and-send -o wide
```

Ensure sufficient nodes or adjust affinity rules in `values.yaml`.

### ConfigMap Not Updating

Force pod restart after ConfigMap changes:

```bash
kubectl rollout restart deployment <deployment-name>
```

(Checksum annotations should handle this automatically)

## Security Considerations

- **Non-root user**: Containers run as UID 1000
- **Read-only root filesystem**: Prevents modification of container filesystem
- **Dropped capabilities**: All Linux capabilities dropped
- **Secret management**: Use `existingSecret` for production deployments
- **Network policies**: Consider adding NetworkPolicy to restrict egress

## Performance Tips

1. **Use emptyDir for speed**: Default storage type, fastest template access
2. **Tune concurrency**: Balance between `concurrency` and `rateLimit.eventsPerSecond`
3. **Scale horizontally**: More pods = better distribution and fault tolerance
4. **Monitor resource usage**: Use `kubectl top` to identify bottlenecks
5. **Adjust batch sizes**: Larger batches = fewer requests but more memory
6. **Pod anti-affinity**: Ensures pods spread across nodes for resilience

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                       Kubernetes Cluster                     │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    Deployment (N replicas)            │  │
│  │                                                        │  │
│  │  ┌──────────────────────────────────────────────┐    │  │
│  │  │ Pod 1                                        │    │  │
│  │  │  ┌────────────┐      ┌──────────────────┐   │    │  │
│  │  │  │ Init:      │      │ Main Container:  │   │    │  │
│  │  │  │ Generator  │─────▶│ Sender           │   │    │  │
│  │  │  │            │      │ (Continuous)     │───┼────┼──▶ OTLP
│  │  │  └────────────┘      └──────────────────┘   │    │  │
│  │  │       │                      │               │    │  │
│  │  │       └──────── emptyDir ────┘               │    │  │
│  │  └──────────────────────────────────────────────┘    │  │
│  │                                                        │  │
│  │  ┌──────────────────────────────────────────────┐    │  │
│  │  │ Pod N (same structure)                       │    │  │
│  │  └──────────────────────────────────────────────┘    │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ConfigMaps: generator.yaml, sender.yaml                    │
│  Secret: Honeycomb API key                                  │
└─────────────────────────────────────────────────────────────┘
```

## Support

For issues, questions, or contributions, please visit:
https://github.com/honeycomb/telemetry-gen-and-send
