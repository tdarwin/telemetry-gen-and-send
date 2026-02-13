package exporter

import (
	"context"
	"fmt"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TraceExporter exports traces via OTLP gRPC
type TraceExporter struct {
	client otlpcollectortrace.TraceServiceClient
	conn   *grpc.ClientConn
	headers map[string]string
}

// NewTraceExporter creates a new trace exporter
func NewTraceExporter(endpoint string, headers map[string]string, insecureConn bool) (*TraceExporter, error) {
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

	client := otlpcollectortrace.NewTraceServiceClient(conn)

	return &TraceExporter{
		client:  client,
		conn:    conn,
		headers: headers,
	}, nil
}

// Export exports a batch of traces
func (e *TraceExporter) Export(ctx context.Context, request *otlpcollectortrace.ExportTraceServiceRequest) error {
	// Add headers to context
	if len(e.headers) > 0 {
		md := metadata.New(e.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	_, err := e.client.Export(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to export traces: %w", err)
	}

	return nil
}

// Close closes the exporter connection
func (e *TraceExporter) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}
