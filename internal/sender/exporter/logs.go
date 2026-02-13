package exporter

import (
	"context"
	"fmt"

	otlpcollectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// LogsExporter exports logs via OTLP gRPC
type LogsExporter struct {
	client  otlpcollectorlogs.LogsServiceClient
	conn    *grpc.ClientConn
	headers map[string]string
}

// NewLogsExporter creates a new logs exporter
func NewLogsExporter(endpoint string, headers map[string]string, insecureConn bool) (*LogsExporter, error) {
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

	client := otlpcollectorlogs.NewLogsServiceClient(conn)

	return &LogsExporter{
		client:  client,
		conn:    conn,
		headers: headers,
	}, nil
}

// Export exports a batch of logs
func (e *LogsExporter) Export(ctx context.Context, request *otlpcollectorlogs.ExportLogsServiceRequest) error {
	// Add headers to context
	if len(e.headers) > 0 {
		md := metadata.New(e.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	_, err := e.client.Export(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to export logs: %w", err)
	}

	return nil
}

// Close closes the exporter connection
func (e *LogsExporter) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}
