package metrics

// MetricType represents the type of metric
type MetricType int

const (
	MetricTypeGauge MetricType = iota
	MetricTypeSum
	MetricTypeHistogram
)

// MetricDefinition defines a metric with its properties
type MetricDefinition struct {
	Name        string
	Description string
	Unit        string
	Type        MetricType
	Dimensions  []string // Dimension keys for this metric
}

// GetHostMetrics returns definitions for host-level metrics
func GetHostMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "system.cpu.utilization",
			Description: "CPU utilization percentage",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"host.name", "os.type", "cpu"},
		},
		{
			Name:        "system.cpu.time",
			Description: "CPU time in seconds",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "os.type", "cpu", "state"},
		},
		{
			Name:        "system.memory.usage",
			Description: "Memory usage in bytes",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"host.name", "os.type", "state"},
		},
		{
			Name:        "system.memory.utilization",
			Description: "Memory utilization percentage",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"host.name", "os.type"},
		},
		{
			Name:        "system.disk.io",
			Description: "Disk I/O bytes",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "device", "direction"},
		},
		{
			Name:        "system.disk.operations",
			Description: "Disk operations",
			Unit:        "{operations}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "device", "direction"},
		},
		{
			Name:        "system.disk.utilization",
			Description: "Disk utilization percentage",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"host.name", "device"},
		},
		{
			Name:        "system.network.io",
			Description: "Network I/O bytes",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "device", "direction"},
		},
		{
			Name:        "system.network.packets",
			Description: "Network packets",
			Unit:        "{packets}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "device", "direction"},
		},
		{
			Name:        "system.network.errors",
			Description: "Network errors",
			Unit:        "{errors}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"host.name", "device", "direction"},
		},
	}
}

// GetK8sClusterMetrics returns definitions for K8s cluster-level metrics
func GetK8sClusterMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "k8s.cluster.node.count",
			Description: "Number of nodes in the cluster",
			Unit:        "{nodes}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name"},
		},
		{
			Name:        "k8s.cluster.pod.count",
			Description: "Number of pods in the cluster",
			Unit:        "{pods}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name"},
		},
	}
}

// GetK8sNodeMetrics returns definitions for K8s node-level metrics
func GetK8sNodeMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "k8s.node.cpu.utilization",
			Description: "Node CPU utilization",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.node.name"},
		},
		{
			Name:        "k8s.node.memory.usage",
			Description: "Node memory usage",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.node.name"},
		},
		{
			Name:        "k8s.node.network.io",
			Description: "Node network I/O",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{"k8s.cluster.name", "k8s.node.name", "direction"},
		},
	}
}

// GetK8sPodMetrics returns definitions for K8s pod-level metrics
func GetK8sPodMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "k8s.pod.cpu.usage",
			Description: "Pod CPU usage",
			Unit:        "{cores}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "k8s.node.name"},
		},
		{
			Name:        "k8s.pod.cpu.limit",
			Description: "Pod CPU limit",
			Unit:        "{cores}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name"},
		},
		{
			Name:        "k8s.pod.memory.usage",
			Description: "Pod memory usage",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "k8s.node.name"},
		},
		{
			Name:        "k8s.pod.memory.limit",
			Description: "Pod memory limit",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name"},
		},
		{
			Name:        "k8s.pod.network.io",
			Description: "Pod network I/O",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "direction"},
		},
	}
}

// GetK8sContainerMetrics returns definitions for K8s container-level metrics
func GetK8sContainerMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "k8s.container.cpu.usage",
			Description: "Container CPU usage",
			Unit:        "{cores}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "container.name"},
		},
		{
			Name:        "k8s.container.cpu.limit",
			Description: "Container CPU limit",
			Unit:        "{cores}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "container.name"},
		},
		{
			Name:        "k8s.container.memory.usage",
			Description: "Container memory usage",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "container.name"},
		},
		{
			Name:        "k8s.container.memory.limit",
			Description: "Container memory limit",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "container.name"},
		},
		{
			Name:        "k8s.container.restarts",
			Description: "Container restart count",
			Unit:        "{restarts}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"k8s.cluster.name", "k8s.namespace.name", "k8s.pod.name", "container.name"},
		},
	}
}

// GetJVMMetrics returns definitions for JVM runtime metrics
func GetJVMMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Memory metrics
		{
			Name:        "jvm.memory.used",
			Description: "Memory used in bytes",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"jvm.memory.type", "jvm.memory.pool.name"},
		},
		{
			Name:        "jvm.memory.committed",
			Description: "Memory committed in bytes",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"jvm.memory.type", "jvm.memory.pool.name"},
		},
		{
			Name:        "jvm.memory.limit",
			Description: "Memory limit in bytes",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"jvm.memory.type", "jvm.memory.pool.name"},
		},
		{
			Name:        "jvm.memory.used_after_last_gc",
			Description: "Memory used after last GC",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"jvm.memory.type", "jvm.memory.pool.name"},
		},
		// GC metrics
		{
			Name:        "jvm.gc.duration",
			Description: "GC duration in milliseconds",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"jvm.gc.name", "jvm.gc.action"},
		},
		// Thread metrics
		{
			Name:        "jvm.thread.count",
			Description: "Number of JVM threads",
			Unit:        "{threads}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"jvm.thread.state", "jvm.thread.daemon"},
		},
		// CPU metrics
		{
			Name:        "jvm.cpu.time",
			Description: "JVM CPU time",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "jvm.cpu.recent_utilization",
			Description: "Recent JVM CPU utilization",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "jvm.cpu.count",
			Description: "Number of CPUs available to JVM",
			Unit:        "{cpus}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		// Class loading metrics
		{
			Name:        "jvm.class.loaded",
			Description: "Number of classes loaded",
			Unit:        "{classes}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "jvm.class.unloaded",
			Description: "Number of classes unloaded",
			Unit:        "{classes}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "jvm.class.count",
			Description: "Current number of loaded classes",
			Unit:        "{classes}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
	}
}

// GetHTTPMetrics returns definitions for HTTP server and client metrics
func GetHTTPMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Server metrics
		{
			Name:        "http.server.duration",
			Description: "HTTP server request duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code", "http.route", "http.scheme"},
		},
		{
			Name:        "http.server.request.duration",
			Description: "HTTP server request duration (semantic conventions v1.21+)",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.request.method", "http.response.status_code", "http.route"},
		},
		{
			Name:        "http.server.active_requests",
			Description: "Number of active HTTP server requests",
			Unit:        "{requests}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"http.method", "http.scheme", "server.address", "server.port"},
		},
		{
			Name:        "http.server.request.size",
			Description: "HTTP server request body size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code", "http.route"},
		},
		{
			Name:        "http.server.response.size",
			Description: "HTTP server response body size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code", "http.route"},
		},
		{
			Name:        "http.server.response.body.size",
			Description: "HTTP server response body size (semantic conventions v1.21+)",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.request.method", "http.response.status_code", "http.route"},
		},
		// Client metrics
		{
			Name:        "http.client.duration",
			Description: "HTTP client request duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code", "http.host"},
		},
		{
			Name:        "http.client.request.size",
			Description: "HTTP client request body size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code"},
		},
		{
			Name:        "http.client.response.size",
			Description: "HTTP client response body size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"http.method", "http.status_code"},
		},
	}
}

// GetApplicationMetrics returns definitions for common application-level metrics
func GetApplicationMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Cart/E-commerce metrics
		{
			Name:        "app.cart.add_item.latency",
			Description: "Add item to cart latency",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"service.name"},
		},
		{
			Name:        "app.cart.get_cart.latency",
			Description: "Get cart latency",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"service.name"},
		},
		// Payment metrics
		{
			Name:        "app.payment.transactions",
			Description: "Payment transactions count",
			Unit:        "{transactions}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"app.payment.currency", "service.name"},
		},
		// Ad metrics
		{
			Name:        "app.ads.ad_requests",
			Description: "Ad requests count",
			Unit:        "{requests}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"app.ads.ad_request_type", "app.ads.ad_response_type", "service.name"},
		},
		// Frontend metrics
		{
			Name:        "app.frontend.requests",
			Description: "Frontend requests count",
			Unit:        "{requests}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"service.name"},
		},
		// Currency counter
		{
			Name:        "app.currency_counter",
			Description: "Currency conversion counter",
			Unit:        "{conversions}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"currency_code", "service.name"},
		},
		// Recommendations counter
		{
			Name:        "app_recommendations_counter",
			Description: "Product recommendations counter",
			Unit:        "{recommendations}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"recommendation.type", "service.name"},
		},
		// Feature flags
		{
			Name:        "feature_flag.flagd.impression",
			Description: "Feature flag evaluations",
			Unit:        "{evaluations}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"feature_flag.key", "feature_flag.variant", "feature_flag.reason"},
		},
		// Queue metrics
		{
			Name:        "app.queue.size",
			Description: "Queue size",
			Unit:        "{items}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"service.name"},
		},
		{
			Name:        "app.queue.processed",
			Description: "Items processed from queue",
			Unit:        "{items}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"service.name", "success"},
		},
	}
}

// GetDatabaseMetrics returns definitions for database and data store metrics
func GetDatabaseMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Generic database metrics
		{
			Name:        "db.client.connections.usage",
			Description: "Number of active connections",
			Unit:        "{connections}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"db.system", "state"},
		},
		{
			Name:        "db.client.connections.max",
			Description: "Maximum number of connections",
			Unit:        "{connections}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"db.system"},
		},
		{
			Name:        "db.client.operation.duration",
			Description: "Database operation duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"db.system", "db.operation", "db.name"},
		},
		// Redis metrics
		{
			Name:        "redis_pubsub_published",
			Description: "Redis pubsub messages published",
			Unit:        "{messages}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"service.name"},
		},
		{
			Name:        "redis_pubsub_received",
			Description: "Redis pubsub messages received",
			Unit:        "{messages}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"service.name"},
		},
	}
}

// GetRPCMetrics returns definitions for RPC/gRPC metrics
func GetRPCMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Server metrics
		{
			Name:        "rpc.server.duration",
			Description: "RPC server call duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method", "rpc.grpc.status_code"},
		},
		{
			Name:        "rpc.server.request.size",
			Description: "RPC server request size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.server.response.size",
			Description: "RPC server response size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.server.requests_per_rpc",
			Description: "Requests per RPC call (streaming)",
			Unit:        "{requests}",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.server.responses_per_rpc",
			Description: "Responses per RPC call (streaming)",
			Unit:        "{responses}",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		// Client metrics
		{
			Name:        "rpc.client.duration",
			Description: "RPC client call duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method", "rpc.grpc.status_code"},
		},
		{
			Name:        "rpc.client.request.size",
			Description: "RPC client request size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.client.response.size",
			Description: "RPC client response size",
			Unit:        "By",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.client.requests_per_rpc",
			Description: "Requests per RPC call (streaming)",
			Unit:        "{requests}",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
		{
			Name:        "rpc.client.responses_per_rpc",
			Description: "Responses per RPC call (streaming)",
			Unit:        "{responses}",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"rpc.system", "rpc.service", "rpc.method"},
		},
	}
}

// GetRuntimeMetrics returns definitions for language runtime metrics
func GetRuntimeMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Process metrics
		{
			Name:        "process.cpu.time",
			Description: "Process CPU time",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{"process.cpu.state"},
		},
		{
			Name:        "process.cpu.utilization",
			Description: "Process CPU utilization",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.memory.usage",
			Description: "Process memory usage",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.memory.virtual",
			Description: "Process virtual memory",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.thread.count",
			Description: "Process thread count",
			Unit:        "{threads}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.open_file_descriptor.count",
			Description: "Open file descriptors",
			Unit:        "{descriptors}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.context_switches",
			Description: "Process context switches",
			Unit:        "{switches}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "runtime.uptime",
			Description: "Runtime uptime",
			Unit:        "ms",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		// Go runtime metrics
		{
			Name:        "process.runtime.go.goroutines",
			Description: "Number of goroutines",
			Unit:        "{goroutines}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.gc.count",
			Description: "Number of GC runs",
			Unit:        "{collections}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.gc.pause_ns",
			Description: "GC pause duration",
			Unit:        "ns",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.mem.heap_alloc",
			Description: "Heap memory allocated",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.mem.heap_idle",
			Description: "Heap memory idle",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.mem.heap_inuse",
			Description: "Heap memory in use",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.go.mem.heap_objects",
			Description: "Number of heap objects",
			Unit:        "{objects}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		// Node.js runtime metrics
		{
			Name:        "nodejs.eventloop.delay.mean",
			Description: "Event loop delay mean",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "nodejs.eventloop.delay.p50",
			Description: "Event loop delay p50",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "nodejs.eventloop.delay.p90",
			Description: "Event loop delay p90",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "nodejs.eventloop.delay.p99",
			Description: "Event loop delay p99",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "nodejs.eventloop.utilization",
			Description: "Event loop utilization",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		// Python runtime metrics
		{
			Name:        "process.runtime.cpython.memory",
			Description: "Python memory usage",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.cpython.gc_count",
			Description: "Python GC count",
			Unit:        "{collections}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.cpython.cpu_time",
			Description: "Python CPU time",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.cpython.thread_count",
			Description: "Python thread count",
			Unit:        "{threads}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		// .NET runtime metrics
		{
			Name:        "process.runtime.dotnet.gc.heap.size",
			Description: ".NET GC heap size",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.gc.collections.count",
			Description: ".NET GC collections",
			Unit:        "{collections}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.gc.duration",
			Description: ".NET GC duration",
			Unit:        "ms",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.thread_pool.threads.count",
			Description: ".NET thread pool thread count",
			Unit:        "{threads}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.thread_pool.queue.length",
			Description: ".NET thread pool queue length",
			Unit:        "{items}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.jit.compilation_time",
			Description: ".NET JIT compilation time",
			Unit:        "ms",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "process.runtime.dotnet.exceptions.count",
			Description: ".NET exceptions count",
			Unit:        "{exceptions}",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
	}
}

// GetMessagingMetrics returns definitions for messaging system metrics
func GetMessagingMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Kafka Consumer metrics
		{
			Name:        "kafka.consumer.records_consumed_total",
			Description: "Total records consumed",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.records_consumed_rate",
			Description: "Records consumed per second",
			Unit:        "{records}/s",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.bytes_consumed_total",
			Description: "Total bytes consumed",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.bytes_consumed_rate",
			Description: "Bytes consumed per second",
			Unit:        "By/s",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.fetch_latency_avg",
			Description: "Average fetch latency",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.fetch_latency_max",
			Description: "Maximum fetch latency",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.records_lag",
			Description: "Consumer lag in records",
			Unit:        "{records}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id", "topic", "partition"},
		},
		{
			Name:        "kafka.consumer.records_lag_max",
			Description: "Maximum consumer lag",
			Unit:        "{records}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.commit_latency_avg",
			Description: "Average commit latency",
			Unit:        "ms",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.assigned_partitions",
			Description: "Number of assigned partitions",
			Unit:        "{partitions}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id", "topic"},
		},
		{
			Name:        "kafka.consumer.connection_count",
			Description: "Number of active connections",
			Unit:        "{connections}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.request_rate",
			Description: "Request rate",
			Unit:        "{requests}/s",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.response_rate",
			Description: "Response rate",
			Unit:        "{responses}/s",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.network_io_rate",
			Description: "Network I/O rate",
			Unit:        "By/s",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
		{
			Name:        "kafka.consumer.io_wait_ratio",
			Description: "I/O wait time ratio",
			Unit:        "%",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"client-id"},
		},
	}
}

// GetOTelCollectorMetrics returns definitions for OpenTelemetry Collector metrics
func GetOTelCollectorMetrics() []MetricDefinition {
	return []MetricDefinition{
		// Receiver metrics
		{
			Name:        "otelcol_receiver_accepted_spans",
			Description: "Spans successfully pushed into the pipeline",
			Unit:        "{spans}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		{
			Name:        "otelcol_receiver_accepted_metric_points",
			Description: "Metric points successfully pushed into the pipeline",
			Unit:        "{datapoints}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		{
			Name:        "otelcol_receiver_accepted_log_records",
			Description: "Log records successfully pushed into the pipeline",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		{
			Name:        "otelcol_receiver_refused_spans",
			Description: "Spans that could not be pushed into the pipeline",
			Unit:        "{spans}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		{
			Name:        "otelcol_receiver_refused_metric_points",
			Description: "Metric points that could not be pushed into the pipeline",
			Unit:        "{datapoints}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		{
			Name:        "otelcol_receiver_refused_log_records",
			Description: "Log records that could not be pushed into the pipeline",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"receiver", "transport"},
		},
		// Processor metrics
		{
			Name:        "otelcol_processor_accepted_spans",
			Description: "Spans successfully pushed into next component",
			Unit:        "{spans}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"processor"},
		},
		{
			Name:        "otelcol_processor_accepted_metric_points",
			Description: "Metric points successfully pushed into next component",
			Unit:        "{datapoints}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"processor"},
		},
		{
			Name:        "otelcol_processor_accepted_log_records",
			Description: "Log records successfully pushed into next component",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"processor"},
		},
		{
			Name:        "otelcol_processor_batch_batch_send_size",
			Description: "Batch processor send batch size",
			Unit:        "{items}",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{"processor"},
		},
		{
			Name:        "otelcol_processor_batch_timeout_trigger_send",
			Description: "Batches sent due to timeout",
			Unit:        "{batches}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"processor"},
		},
		{
			Name:        "otelcol_processor_batch_metadata_cardinality",
			Description: "Batch metadata cardinality",
			Unit:        "{count}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"processor"},
		},
		// Exporter metrics
		{
			Name:        "otelcol_exporter_sent_spans",
			Description: "Spans successfully sent to destination",
			Unit:        "{spans}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_sent_metric_points",
			Description: "Metric points successfully sent to destination",
			Unit:        "{datapoints}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_sent_log_records",
			Description: "Log records successfully sent to destination",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_send_failed_spans",
			Description: "Spans failed to send to destination",
			Unit:        "{spans}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_send_failed_metric_points",
			Description: "Metric points failed to send to destination",
			Unit:        "{datapoints}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_send_failed_log_records",
			Description: "Log records failed to send to destination",
			Unit:        "{records}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_queue_size",
			Description: "Current size of the retry queue",
			Unit:        "{items}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"exporter"},
		},
		{
			Name:        "otelcol_exporter_queue_capacity",
			Description: "Fixed capacity of the retry queue",
			Unit:        "{items}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{"exporter"},
		},
		// Process metrics
		{
			Name:        "otelcol_process_uptime",
			Description: "Uptime of the collector",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "otelcol_process_runtime_heap_alloc_bytes",
			Description: "Bytes of allocated heap objects",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "otelcol_process_runtime_total_alloc_bytes",
			Description: "Cumulative bytes allocated for heap objects",
			Unit:        "By",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
		{
			Name:        "otelcol_process_runtime_total_sys_memory_bytes",
			Description: "Total bytes of memory obtained from OS",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "otelcol_process_memory_rss",
			Description: "Resident set size",
			Unit:        "By",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "otelcol_process_cpu_seconds",
			Description: "Total CPU seconds",
			Unit:        "s",
			Type:        MetricTypeSum,
			Dimensions:  []string{},
		},
	}
}

// GetASPNETMetrics returns definitions for ASP.NET Core / Kestrel metrics
func GetASPNETMetrics() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:        "kestrel.active_connections",
			Description: "Number of active connections",
			Unit:        "{connections}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "kestrel.connection.duration",
			Description: "Connection duration",
			Unit:        "ms",
			Type:        MetricTypeHistogram,
			Dimensions:  []string{},
		},
		{
			Name:        "kestrel.queued_connections",
			Description: "Number of queued connections",
			Unit:        "{connections}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "kestrel.queued_requests",
			Description: "Number of queued requests",
			Unit:        "{requests}",
			Type:        MetricTypeGauge,
			Dimensions:  []string{},
		},
		{
			Name:        "aspnetcore.routing.match_attempts",
			Description: "Route matching attempts",
			Unit:        "{attempts}",
			Type:        MetricTypeSum,
			Dimensions:  []string{"aspnetcore.routing.match_status"},
		},
	}
}

// GetMetricsByType returns metric definitions for a given type
func GetMetricsByType(metricType string) []MetricDefinition {
	switch metricType {
	case "host_metrics":
		return GetHostMetrics()
	case "k8s_cluster":
		return GetK8sClusterMetrics()
	case "k8s_node":
		return GetK8sNodeMetrics()
	case "k8s_pod":
		return GetK8sPodMetrics()
	case "k8s_container":
		return GetK8sContainerMetrics()
	case "jvm_metrics":
		return GetJVMMetrics()
	case "http_metrics":
		return GetHTTPMetrics()
	case "application_metrics":
		return GetApplicationMetrics()
	case "database_metrics":
		return GetDatabaseMetrics()
	case "rpc_metrics":
		return GetRPCMetrics()
	case "runtime_metrics":
		return GetRuntimeMetrics()
	case "messaging_metrics":
		return GetMessagingMetrics()
	case "otelcol_metrics":
		return GetOTelCollectorMetrics()
	case "aspnet_metrics":
		return GetASPNETMetrics()
	default:
		return []MetricDefinition{}
	}
}

// GetAllMetrics returns all metric definitions for the given types
func GetAllMetrics(types []string) []MetricDefinition {
	allMetrics := make([]MetricDefinition, 0)

	for _, metricType := range types {
		metrics := GetMetricsByType(metricType)
		allMetrics = append(allMetrics, metrics...)
	}

	return allMetrics
}

// SelectMetrics selects a subset of metrics up to the target count
func SelectMetrics(allMetrics []MetricDefinition, targetCount int) []MetricDefinition {
	if targetCount >= len(allMetrics) {
		return allMetrics
	}

	// If we need fewer metrics, select evenly from all types
	selected := make([]MetricDefinition, 0, targetCount)
	step := float64(len(allMetrics)) / float64(targetCount)

	for i := 0; i < targetCount; i++ {
		index := int(float64(i) * step)
		if index >= len(allMetrics) {
			index = len(allMetrics) - 1
		}
		selected = append(selected, allMetrics[index])
	}

	return selected
}

// ToOTLPMetricType converts our MetricType to OTLP metric data type
func (m MetricType) ToOTLPMetricType() string {
	switch m {
	case MetricTypeGauge:
		return "Gauge"
	case MetricTypeSum:
		return "Sum"
	case MetricTypeHistogram:
		return "Histogram"
	default:
		return "Gauge"
	}
}

// String returns a string representation of the metric type
func (m MetricType) String() string {
	return m.ToOTLPMetricType()
}

// GetValueRange returns appropriate value ranges for different metric types
func (d *MetricDefinition) GetValueRange() (float64, float64) {
	switch {
	// Percentage metrics
	case d.Unit == "%":
		return 0.0, 100.0

	// Time metrics
	case d.Unit == "ms":
		return 1.0, 5000.0 // 1ms to 5s
	case d.Unit == "s":
		return 0.1, 300.0 // 0.1s to 5min
	case d.Unit == "ns":
		return 1000.0, 1e9 // 1μs to 1s

	// Memory metrics
	case d.Unit == "By" && (d.Name == "system.memory.usage" || d.Name == "jvm.memory.used"):
		return 1e8, 4e9 // 100MB to 4GB
	case d.Unit == "By" && d.Name == "jvm.memory.limit":
		return 5e8, 8e9 // 500MB to 8GB
	case d.Unit == "By" && d.Name == "process.memory.usage":
		return 1e7, 5e8 // 10MB to 500MB
	case d.Unit == "By":
		return 100, 1e6 // 100 bytes to 1MB for other byte metrics (request/response sizes)

	// Rate metrics
	case d.Unit == "{records}/s" || d.Unit == "{requests}/s" || d.Unit == "{responses}/s":
		return 0, 10000
	case d.Unit == "By/s":
		return 0, 1e8 // 0 to 100MB/s

	// Count metrics
	case d.Unit == "{threads}":
		return 1, 500
	case d.Unit == "{goroutines}":
		return 10, 10000
	case d.Unit == "{connections}":
		return 0, 1000
	case d.Unit == "{requests}" || d.Unit == "{responses}":
		return 1, 100
	case d.Unit == "{items}":
		return 0, 10000
	case d.Unit == "{classes}":
		return 100, 50000
	case d.Unit == "{cpus}":
		return 1, 64
	case d.Unit == "{descriptors}":
		return 10, 10000
	case d.Unit == "{transactions}":
		return 0, 1000
	case d.Unit == "{conversions}":
		return 0, 500
	case d.Unit == "{recommendations}":
		return 0, 100
	case d.Unit == "{evaluations}":
		return 0, 10000
	case d.Unit == "{messages}":
		return 0, 100000
	case d.Unit == "{collections}":
		return 0, 1000
	case d.Unit == "{exceptions}":
		return 0, 100
	case d.Unit == "{switches}":
		return 0, 100000
	case d.Unit == "{objects}":
		return 1000, 1000000
	case d.Unit == "{records}":
		return 0, 1000000
	case d.Unit == "{datapoints}":
		return 0, 100000
	case d.Unit == "{spans}":
		return 0, 100000
	case d.Unit == "{partitions}":
		return 1, 100
	case d.Unit == "{batches}":
		return 0, 1000
	case d.Unit == "{count}":
		return 0, 1000
	case d.Unit == "{attempts}":
		return 0, 10000

	// K8s specific
	case d.Name == "k8s.container.restarts":
		return 0, 10
	case d.Unit == "{cores}":
		return 0.1, 4.0
	case d.Unit == "{nodes}":
		return 3, 100
	case d.Unit == "{pods}":
		return 10, 500

	// Default
	default:
		return 0, 1000
	}
}
