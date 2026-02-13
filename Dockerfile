# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION} -s -w" -o /telemetry-generator ./cmd/telemetry-generator && \
    CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION} -s -w" -o /telemetry-sender ./cmd/telemetry-sender

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 telemetry && \
    adduser -D -u 1000 -G telemetry telemetry

# Create directories
RUN mkdir -p /data /config && \
    chown -R telemetry:telemetry /data /config

# Copy binaries from builder
COPY --from=builder /telemetry-generator /usr/local/bin/telemetry-generator
COPY --from=builder /telemetry-sender /usr/local/bin/telemetry-sender

# Copy example configs
COPY --chown=telemetry:telemetry examples/ /config/examples/

# Switch to non-root user
USER telemetry

# Set working directory
WORKDIR /data

# Default command (can be overridden)
CMD ["telemetry-generator", "--help"]

# Labels
LABEL org.opencontainers.image.title="Telemetry Generator and Sender"
LABEL org.opencontainers.image.description="High-performance OpenTelemetry trace, metric, and log generator for load testing"
LABEL org.opencontainers.image.vendor="Honeycomb"
LABEL org.opencontainers.image.source="https://github.com/honeycomb/telemetry-gen-and-send"
