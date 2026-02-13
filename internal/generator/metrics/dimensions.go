package metrics

import (
	"fmt"

	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
	"go.opentelemetry.io/proto/otlp/common/v1"
)

// DimensionSet represents a unique combination of dimension values
type DimensionSet map[string]string

// DimensionGenerator generates dimension combinations for metrics
type DimensionGenerator struct {
	hostnameGen    *common.HostnameGenerator
	podGen         map[string]*common.PodNameGenerator
	containerGen   *common.ContainerNameGenerator
	nodeGen        *common.NodeNameGenerator
	namespaceGen   *common.NamespaceGenerator
	regionGen      *common.RegionGenerator
	osTypeGen      *common.OSTypeGenerator
	clusterName    string
}

// NewDimensionGenerator creates a new dimension generator
func NewDimensionGenerator() *DimensionGenerator {
	clusterName := common.GenerateClusterName()

	return &DimensionGenerator{
		hostnameGen:  common.NewHostnameGenerator(),
		podGen:       make(map[string]*common.PodNameGenerator),
		containerGen: common.NewContainerNameGenerator(),
		nodeGen:      common.NewNodeNameGenerator(clusterName),
		namespaceGen: common.NewNamespaceGenerator(),
		regionGen:    common.NewRegionGenerator(),
		osTypeGen:    common.NewOSTypeGenerator(),
		clusterName:  clusterName,
	}
}

// GenerateDimensionSets generates N unique dimension sets for a metric
func (g *DimensionGenerator) GenerateDimensionSets(metric MetricDefinition, count int) []DimensionSet {
	sets := make([]DimensionSet, 0, count)

	for i := 0; i < count; i++ {
		set := g.generateSingleSet(metric)
		sets = append(sets, set)
	}

	return sets
}

// generateSingleSet generates a single dimension set
func (g *DimensionGenerator) generateSingleSet(metric MetricDefinition) DimensionSet {
	set := make(DimensionSet)

	for _, dimKey := range metric.Dimensions {
		set[dimKey] = g.generateDimensionValue(dimKey)
	}

	return set
}

// generateDimensionValue generates a value for a specific dimension key
func (g *DimensionGenerator) generateDimensionValue(key string) string {
	switch key {
	case "host.name":
		return g.hostnameGen.Generate()

	case "os.type":
		return g.osTypeGen.Generate()

	case "cpu":
		return fmt.Sprintf("cpu%d", common.RandomInt(0, 7))

	case "state":
		// Memory or CPU state
		states := []string{"used", "free", "cached", "buffered", "idle", "system", "user", "iowait"}
		return common.RandomChoice(states)

	case "device":
		// Disk or network device
		devices := []string{"sda", "sda1", "sda2", "nvme0n1", "eth0", "eth1", "lo"}
		return common.RandomChoice(devices)

	case "direction":
		directions := []string{"read", "write", "transmit", "receive"}
		return common.RandomChoice(directions)

	case "k8s.cluster.name":
		return g.clusterName

	case "k8s.namespace.name":
		return g.namespaceGen.Generate()

	case "k8s.pod.name":
		// Use namespace as key to generate consistent pod names per namespace
		namespace := "default"
		if podGen, ok := g.podGen[namespace]; ok {
			return podGen.Generate()
		}
		// Create new pod generator for this namespace
		deployment := common.GenerateDeploymentName(fmt.Sprintf("app-%d", len(g.podGen)))
		podGen := common.NewPodNameGenerator(deployment)
		g.podGen[namespace] = podGen
		return podGen.Generate()

	case "k8s.node.name":
		return g.nodeGen.Generate()

	case "container.name":
		return g.containerGen.Generate()

	case "cloud.provider":
		providers := []string{"aws", "gcp", "azure"}
		return common.RandomChoice(providers)

	case "cloud.region":
		return g.regionGen.Generate()

	case "cloud.availability_zone":
		region := g.regionGen.Generate()
		azGen := common.NewAvailabilityZoneGenerator(region)
		return azGen.Generate()

	default:
		// Unknown dimension, generate generic value
		return fmt.Sprintf("value-%d", common.RandomInt(1, 100))
	}
}

// ToAttributes converts a DimensionSet to OTLP attributes
func (ds DimensionSet) ToAttributes() []*v1.KeyValue {
	attrs := make([]*v1.KeyValue, 0, len(ds))

	for key, value := range ds {
		attrs = append(attrs, &v1.KeyValue{
			Key: key,
			Value: &v1.AnyValue{
				Value: &v1.AnyValue_StringValue{
					StringValue: value,
				},
			},
		})
	}

	return attrs
}

// String returns a string representation of the dimension set
func (ds DimensionSet) String() string {
	result := "{"
	first := true
	for key, value := range ds {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%s", key, value)
		first = false
	}
	result += "}"
	return result
}

// Clone creates a copy of the dimension set
func (ds DimensionSet) Clone() DimensionSet {
	clone := make(DimensionSet)
	for k, v := range ds {
		clone[k] = v
	}
	return clone
}

// Merge merges another dimension set into this one
func (ds DimensionSet) Merge(other DimensionSet) {
	for k, v := range other {
		ds[k] = v
	}
}
