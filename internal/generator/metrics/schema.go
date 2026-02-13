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
	case d.Unit == "%":
		return 0.0, 100.0
	case d.Unit == "By" && d.Name == "system.memory.usage":
		return 1e9, 16e9 // 1GB to 16GB
	case d.Unit == "By":
		return 0, 1e9 // 0 to 1GB for other byte metrics
	case d.Name == "k8s.container.restarts":
		return 0, 10
	case d.Unit == "{cores}":
		return 0.1, 4.0
	case d.Unit == "{nodes}":
		return 3, 100
	case d.Unit == "{pods}":
		return 10, 500
	default:
		return 0, 1000
	}
}
