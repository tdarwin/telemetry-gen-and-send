package logs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/honeycomb/telemetry-gen-and-send/internal/config"
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
	otlpcollectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	otlplogs "go.opentelemetry.io/proto/otlp/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

// Generator is the main logs generator
type Generator struct {
	config       *config.LogsConfig
	serviceNames []string
	outputDir    string
	prefix       string
}

// NewGenerator creates a new logs generator
func NewGenerator(cfg *config.LogsConfig, outputDir, prefix string) *Generator {
	// Generate service names for application logs
	serviceNames := make([]string, cfg.Types.Application.Services)
	for i := 0; i < cfg.Types.Application.Services; i++ {
		serviceNames[i] = fmt.Sprintf("app-service-%d", i+1)
	}

	return &Generator{
		config:       cfg,
		serviceNames: serviceNames,
		outputDir:    outputDir,
		prefix:       prefix,
	}
}

// Generate generates all logs according to configuration
func (g *Generator) Generate(writeJSON bool) error {
	fmt.Println("Generating logs...")
	fmt.Printf("  Target log count: %d\n", g.config.Count)
	fmt.Printf("  HTTP Access: %d%%\n", g.config.Types.HTTPAccess.Percentage)
	fmt.Printf("  Application: %d%%\n", g.config.Types.Application.Percentage)
	fmt.Printf("  System: %d%%\n", g.config.Types.System.Percentage)

	// Calculate counts per type
	httpCount := (g.config.Count * g.config.Types.HTTPAccess.Percentage) / 100
	appCount := (g.config.Count * g.config.Types.Application.Percentage) / 100
	sysCount := g.config.Count - httpCount - appCount // Remainder goes to system

	logs := make([]*LogTemplate, 0, g.config.Count)

	// Generate HTTP access logs
	fmt.Printf("Generating %d HTTP access logs...\n", httpCount)
	for i := 0; i < httpCount; i++ {
		logs = append(logs, GenerateHTTPAccessLog())
	}

	// Generate application logs
	fmt.Printf("Generating %d application logs...\n", appCount)
	for i := 0; i < appCount; i++ {
		service := common.RandomChoice(g.serviceNames)
		severity := common.RandomLogLevel()
		logs = append(logs, GenerateApplicationLog(service, severity))
	}

	// Generate system logs
	fmt.Printf("Generating %d system logs...\n", sysCount)
	for i := 0; i < sysCount; i++ {
		logs = append(logs, GenerateSystemLog())
	}

	// Print statistics
	sevCounts := g.calculateSeverityCounts(logs)
	fmt.Printf("\nLog Generation Statistics:\n")
	fmt.Printf("  Total logs: %d\n", len(logs))
	fmt.Printf("  HTTP Access: %d\n", httpCount)
	fmt.Printf("  Application: %d\n", appCount)
	fmt.Printf("  System: %d\n", sysCount)
	fmt.Printf("  Severity distribution:\n")
	for sev, count := range sevCounts {
		fmt.Printf("    %s: %d (%.1f%%)\n", sev, count, float64(count)*100/float64(len(logs)))
	}

	// Write to disk
	fmt.Println("\nWriting logs to disk...")
	if err := g.writeLogs(logs, writeJSON); err != nil {
		return fmt.Errorf("failed to write logs: %w", err)
	}

	fmt.Println("✓ Logs generation complete")
	return nil
}

// calculateSeverityCounts counts logs by severity
func (g *Generator) calculateSeverityCounts(logs []*LogTemplate) map[string]int {
	counts := make(map[string]int)
	for _, log := range logs {
		counts[log.Severity]++
	}
	return counts
}

// writeLogs writes logs to protobuf and optionally JSON
func (g *Generator) writeLogs(logs []*LogTemplate, writeJSON bool) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert to OTLP format
	request := g.logsToOTLP(logs)

	// Write protobuf
	pbPath := filepath.Join(g.outputDir, fmt.Sprintf("%s-logs.pb", g.prefix))
	if err := g.writeProtobuf(request, pbPath); err != nil {
		return fmt.Errorf("failed to write protobuf: %w", err)
	}

	fmt.Printf("Wrote %d logs to %s\n", len(logs), pbPath)

	// Write JSON if requested
	if writeJSON {
		jsonPath := filepath.Join(g.outputDir, fmt.Sprintf("%s-logs.json", g.prefix))
		if err := g.writeJSON(request, jsonPath); err != nil {
			return fmt.Errorf("failed to write JSON: %w", err)
		}
		fmt.Printf("Wrote logs JSON to %s\n", jsonPath)
	}

	return nil
}

// logsToOTLP converts log templates to OTLP ExportLogsServiceRequest
func (g *Generator) logsToOTLP(logs []*LogTemplate) *otlpcollectorlogs.ExportLogsServiceRequest {
	request := &otlpcollectorlogs.ExportLogsServiceRequest{
		ResourceLogs: []*otlplogs.ResourceLogs{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key: "service.name",
							Value: &commonpb.AnyValue{
								Value: &commonpb.AnyValue_StringValue{
									StringValue: "telemetry-generator",
								},
							},
						},
					},
				},
				ScopeLogs: []*otlplogs.ScopeLogs{
					{
						Scope: &commonpb.InstrumentationScope{
							Name:    "telemetry-generator",
							Version: "1.0.0",
						},
						LogRecords: make([]*otlplogs.LogRecord, 0),
					},
				},
			},
		},
	}

	// Convert each log template
	for _, logTemplate := range logs {
		logRecord := g.templateToOTLP(logTemplate)
		request.ResourceLogs[0].ScopeLogs[0].LogRecords = append(
			request.ResourceLogs[0].ScopeLogs[0].LogRecords,
			logRecord,
		)
	}

	return request
}

// templateToOTLP converts a log template to OTLP LogRecord
func (g *Generator) templateToOTLP(template *LogTemplate) *otlplogs.LogRecord {
	record := &otlplogs.LogRecord{
		TimeUnixNano:         0, // No timestamp in template
		ObservedTimeUnixNano: 0, // No timestamp in template
		SeverityNumber:       g.severityToNumber(template.Severity),
		SeverityText:         template.Severity,
		Body: &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{
				StringValue: template.Body,
			},
		},
		Attributes: make([]*commonpb.KeyValue, 0),
	}

	// Add attributes
	for key, value := range template.Attributes {
		record.Attributes = append(record.Attributes, g.createAttribute(key, value))
	}

	// Add log type attribute
	record.Attributes = append(record.Attributes, &commonpb.KeyValue{
		Key: "log.type",
		Value: &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{
				StringValue: template.Type.String(),
			},
		},
	})

	return record
}

// createAttribute creates an OTLP KeyValue from a Go value
func (g *Generator) createAttribute(key string, value interface{}) *commonpb.KeyValue {
	kv := &commonpb.KeyValue{Key: key}

	switch v := value.(type) {
	case string:
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: v},
		}
	case int:
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_IntValue{IntValue: int64(v)},
		}
	case int64:
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_IntValue{IntValue: v},
		}
	case float64:
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_DoubleValue{DoubleValue: v},
		}
	case bool:
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_BoolValue{BoolValue: v},
		}
	default:
		// Default to string representation
		kv.Value = &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", v)},
		}
	}

	return kv
}

// severityToNumber converts severity text to OTLP severity number
func (g *Generator) severityToNumber(severity string) otlplogs.SeverityNumber {
	switch severity {
	case "DEBUG":
		return otlplogs.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "INFO":
		return otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO
	case "WARN":
		return otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN
	case "ERROR":
		return otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR
	default:
		return otlplogs.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
	}
}

// writeProtobuf writes the OTLP request as protobuf binary
func (g *Generator) writeProtobuf(request *otlpcollectorlogs.ExportLogsServiceRequest, path string) error {
	data, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// writeJSON writes the OTLP request as JSON
func (g *Generator) writeJSON(request *otlpcollectorlogs.ExportLogsServiceRequest, path string) error {
	data, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
