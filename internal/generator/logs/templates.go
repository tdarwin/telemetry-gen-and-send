package logs

import (
	"fmt"

	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
)

// LogType represents the type of log
type LogType int

const (
	LogTypeHTTPAccess LogType = iota
	LogTypeApplication
	LogTypeSystem
)

// LogTemplate represents a log entry template
type LogTemplate struct {
	Type       LogType
	Severity   string
	Body       string
	Attributes map[string]interface{}
}

// GenerateHTTPAccessLog generates an HTTP access log
func GenerateHTTPAccessLog() *LogTemplate {
	method := common.RandomHTTPMethod()
	path := common.RandomHTTPPath()
	status := common.RandomHTTPStatus()
	responseSize := common.RandomInt(100, 50000)
	duration := common.RandomFloat64(1.0, 500.0) // ms

	// Generate message in Apache Common Log format style
	body := fmt.Sprintf("%s %s %d %d %.2fms",
		method, path, status, responseSize, duration)

	return &LogTemplate{
		Type:     LogTypeHTTPAccess,
		Severity: "INFO",
		Body:     body,
		Attributes: map[string]interface{}{
			"http.method":                 method,
			"http.target":                 path,
			"http.status_code":            status,
			"http.response_content_length": responseSize,
			"http.response_time_ms":       duration,
			"http.user_agent":             "Mozilla/5.0 (compatible)",
		},
	}
}

// GenerateApplicationLog generates an application log
func GenerateApplicationLog(serviceName, severity string) *LogTemplate {
	log := &LogTemplate{
		Type:     LogTypeApplication,
		Severity: severity,
		Attributes: map[string]interface{}{
			"service.name": serviceName,
		},
	}

	// Generate message based on severity
	switch severity {
	case "DEBUG":
		log.Body = generateDebugMessage()

	case "INFO":
		log.Body = generateInfoMessage()

	case "WARN":
		log.Body = generateWarnMessage()
		log.Attributes["warning.type"] = common.RandomChoice([]string{
			"DeprecationWarning",
			"PerformanceWarning",
			"ConfigurationWarning",
		})

	case "ERROR":
		log.Body = generateErrorMessage()
		errorType := common.RandomErrorType()
		log.Attributes["error.type"] = errorType
		log.Attributes["error.message"] = log.Body
		log.Attributes["error.stack"] = generateStackTrace(errorType)
	}

	return log
}

// GenerateSystemLog generates a system log
func GenerateSystemLog() *LogTemplate {
	eventTypes := []string{
		"startup",
		"shutdown",
		"configuration_change",
		"resource_alert",
		"health_check",
		"deployment",
	}

	eventType := common.RandomChoice(eventTypes)
	severity := "INFO"

	var body string
	switch eventType {
	case "startup":
		body = fmt.Sprintf("Service started successfully on port %d", common.RandomInt(3000, 9000))

	case "shutdown":
		body = "Service shutting down gracefully"
		severity = "WARN"

	case "configuration_change":
		body = "Configuration reloaded from /etc/app/config.yaml"

	case "resource_alert":
		resource := common.RandomChoice([]string{"CPU", "Memory", "Disk"})
		usage := common.RandomInt(75, 95)
		body = fmt.Sprintf("%s usage at %d%%, approaching threshold", resource, usage)
		severity = "WARN"

	case "health_check":
		status := common.RandomChoice([]string{"passed", "failed"})
		if status == "failed" {
			body = "Health check failed: database connection timeout"
			severity = "ERROR"
		} else {
			body = "Health check passed"
		}

	case "deployment":
		version := fmt.Sprintf("v1.%d.%d", common.RandomInt(0, 10), common.RandomInt(0, 20))
		body = fmt.Sprintf("Deployed version %s", version)
	}

	return &LogTemplate{
		Type:     LogTypeSystem,
		Severity: severity,
		Body:     body,
		Attributes: map[string]interface{}{
			"log.source": "system",
			"event.type": eventType,
		},
	}
}

// generateDebugMessage generates a debug-level message
func generateDebugMessage() string {
	messages := []string{
		"Processing request with ID: %s",
		"Cache hit for key: %s",
		"Query executed in %.2fms",
		"Connection pool size: %d",
		"Background job started: %s",
	}

	msg := common.RandomChoice(messages)

	switch {
	case msg == messages[0]:
		return fmt.Sprintf(msg, common.RandomString(16))
	case msg == messages[1]:
		return fmt.Sprintf(msg, common.RandomString(8))
	case msg == messages[2]:
		return fmt.Sprintf(msg, common.RandomFloat64(1.0, 50.0))
	case msg == messages[3]:
		return fmt.Sprintf(msg, common.RandomInt(5, 50))
	case msg == messages[4]:
		return fmt.Sprintf(msg, common.RandomChoice([]string{"cleanup", "sync", "backup"}))
	}

	return msg
}

// generateInfoMessage generates an info-level message
func generateInfoMessage() string {
	messages := []string{
		"Request processed successfully",
		"User %s logged in",
		"Order %s created",
		"Payment processed for amount $%.2f",
		"Email sent to %s",
		"Cache cleared",
		"Database migration completed",
		"Report generated: %s",
	}

	msg := common.RandomChoice(messages)

	switch {
	case msg == messages[1]:
		return fmt.Sprintf(msg, fmt.Sprintf("user_%d", common.RandomInt(1, 10000)))
	case msg == messages[2]:
		return fmt.Sprintf(msg, fmt.Sprintf("ORD-%d", common.RandomInt(10000, 99999)))
	case msg == messages[3]:
		return fmt.Sprintf(msg, common.RandomFloat64(10.0, 1000.0))
	case msg == messages[4]:
		return fmt.Sprintf(msg, fmt.Sprintf("user%d@example.com", common.RandomInt(1, 1000)))
	case msg == messages[7]:
		return fmt.Sprintf(msg, fmt.Sprintf("report-%s.pdf", common.RandomString(8)))
	}

	return msg
}

// generateWarnMessage generates a warning-level message
func generateWarnMessage() string {
	messages := []string{
		"Retry attempt %d for operation %s",
		"Deprecated API endpoint /api/v1/%s called",
		"Slow query detected: %.2fms",
		"Rate limit approaching for user %s",
		"Cache miss rate above threshold: %.1f%%",
		"Connection pool exhausted, queueing requests",
	}

	msg := common.RandomChoice(messages)

	switch {
	case msg == messages[0]:
		return fmt.Sprintf(msg, common.RandomInt(1, 3), common.RandomChoice([]string{"save", "update", "delete"}))
	case msg == messages[1]:
		return fmt.Sprintf(msg, common.RandomChoice([]string{"users", "orders", "products"}))
	case msg == messages[2]:
		return fmt.Sprintf(msg, common.RandomFloat64(500.0, 2000.0))
	case msg == messages[3]:
		return fmt.Sprintf(msg, fmt.Sprintf("user_%d", common.RandomInt(1, 1000)))
	case msg == messages[4]:
		return fmt.Sprintf(msg, common.RandomFloat64(40.0, 80.0))
	}

	return msg
}

// generateErrorMessage generates an error-level message
func generateErrorMessage() string {
	messages := []string{
		"Failed to connect to database: connection timeout",
		"Validation failed: invalid email format",
		"Payment processing failed: insufficient funds",
		"File not found: /var/log/app.log",
		"Authentication failed for user %s",
		"API request failed: 503 Service Unavailable",
		"Failed to parse configuration file",
		"Deadlock detected in transaction",
	}

	msg := common.RandomChoice(messages)

	if msg == messages[4] {
		return fmt.Sprintf(msg, fmt.Sprintf("user_%d", common.RandomInt(1, 1000)))
	}

	return msg
}

// generateStackTrace generates a simple stack trace
func generateStackTrace(errorType string) string {
	frames := []string{
		"  at handleRequest (server.go:142)",
		"  at processOrder (orders.go:87)",
		"  at validatePayment (payment.go:234)",
		"  at main (main.go:45)",
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s",
		frames[0], frames[1], frames[2], frames[3])
}

// String returns the log type as a string
func (t LogType) String() string {
	switch t {
	case LogTypeHTTPAccess:
		return "http_access"
	case LogTypeApplication:
		return "application"
	case LogTypeSystem:
		return "system"
	default:
		return "unknown"
	}
}
