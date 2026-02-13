package exporter

import (
	"context"
	"fmt"

	otlpcollectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// MetricsExporter exports metrics via OTLP gRPC
type MetricsExporter struct {
	client  otlpcollectormetrics.MetricsServiceClient
	conn    *grpc.ClientConn
	headers map[string]string
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(endpoint string, headers map[string]string, insecureConn bool) (*MetricsExporter, error) {
	var opts []grpc.DialOption

	if insecureConn {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}

	client := otlpcollectormetrics.NewMetricsServiceClient(conn)

	return &MetricsExporter{
		client:  client,
		conn:    conn,
		headers: headers,
	}, nil
}

// Export exports a batch of metrics
func (e *MetricsExporter) Export(ctx context.Context, request *otlpcollectormetrics.ExportMetricsServiceRequest) error {
	// Add headers to context
	if len(e.headers) > 0 {
		md := metadata.New(e.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	_, err := e.client.Export(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to export metrics: %w", err)
	}

	return nil
}

// Close closes the exporter connection
func (e *MetricsExporter) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}
