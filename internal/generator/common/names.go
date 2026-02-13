package common

import (
	"fmt"
)

// HostnameGenerator generates realistic hostnames
type HostnameGenerator struct {
	counter int
}

// NewHostnameGenerator creates a new hostname generator
func NewHostnameGenerator() *HostnameGenerator {
	return &HostnameGenerator{counter: 0}
}

// Generate generates a hostname
func (g *HostnameGenerator) Generate() string {
	g.counter++
	prefixes := []string{"web", "api", "db", "cache", "worker", "app"}
	prefix := RandomChoice(prefixes)
	return fmt.Sprintf("%s-server-%03d", prefix, g.counter)
}

// PodNameGenerator generates Kubernetes pod names
type PodNameGenerator struct {
	deploymentName string
	counter        int
}

// NewPodNameGenerator creates a new pod name generator
func NewPodNameGenerator(deploymentName string) *PodNameGenerator {
	return &PodNameGenerator{
		deploymentName: deploymentName,
		counter:        0,
	}
}

// Generate generates a pod name
func (g *PodNameGenerator) Generate() string {
	g.counter++
	hash := RandomString(10)
	return fmt.Sprintf("%s-%s", g.deploymentName, hash)
}

// ContainerNameGenerator generates container names
type ContainerNameGenerator struct {
	names []string
	index int
}

// NewContainerNameGenerator creates a new container name generator
func NewContainerNameGenerator() *ContainerNameGenerator {
	names := []string{
		"app",
		"sidecar",
		"init",
		"proxy",
		"metrics",
		"logs",
	}
	return &ContainerNameGenerator{names: names, index: 0}
}

// Generate generates a container name
func (g *ContainerNameGenerator) Generate() string {
	if g.index >= len(g.names) {
		g.index = 0
	}
	name := g.names[g.index]
	g.index++
	return name
}

// NodeNameGenerator generates Kubernetes node names
type NodeNameGenerator struct {
	clusterName string
	counter     int
}

// NewNodeNameGenerator creates a new node name generator
func NewNodeNameGenerator(clusterName string) *NodeNameGenerator {
	return &NodeNameGenerator{
		clusterName: clusterName,
		counter:     0,
	}
}

// Generate generates a node name
func (g *NodeNameGenerator) Generate() string {
	g.counter++
	return fmt.Sprintf("%s-node-%03d", g.clusterName, g.counter)
}

// NamespaceGenerator generates Kubernetes namespace names
type NamespaceGenerator struct {
	namespaces []string
	index      int
}

// NewNamespaceGenerator creates a new namespace generator
func NewNamespaceGenerator() *NamespaceGenerator {
	namespaces := []string{
		"default",
		"production",
		"staging",
		"development",
		"monitoring",
		"logging",
		"kube-system",
	}
	return &NamespaceGenerator{namespaces: namespaces, index: 0}
}

// Generate generates a namespace name
func (g *NamespaceGenerator) Generate() string {
	if g.index >= len(g.namespaces) {
		g.index = 0
	}
	name := g.namespaces[g.index]
	g.index++
	return name
}

// RegionGenerator generates cloud region names
type RegionGenerator struct {
	regions []string
	index   int
}

// NewRegionGenerator creates a new region generator
func NewRegionGenerator() *RegionGenerator {
	regions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"eu-west-1",
		"eu-central-1",
		"ap-southeast-1",
		"ap-northeast-1",
	}
	return &RegionGenerator{regions: regions, index: 0}
}

// Generate generates a region name
func (g *RegionGenerator) Generate() string {
	if g.index >= len(g.regions) {
		g.index = 0
	}
	region := g.regions[g.index]
	g.index++
	return region
}

// AvailabilityZoneGenerator generates availability zone names
type AvailabilityZoneGenerator struct {
	region string
	zones  []string
	index  int
}

// NewAvailabilityZoneGenerator creates a new AZ generator
func NewAvailabilityZoneGenerator(region string) *AvailabilityZoneGenerator {
	zones := []string{"a", "b", "c"}
	return &AvailabilityZoneGenerator{
		region: region,
		zones:  zones,
		index:  0,
	}
}

// Generate generates an availability zone name
func (g *AvailabilityZoneGenerator) Generate() string {
	if g.index >= len(g.zones) {
		g.index = 0
	}
	zone := fmt.Sprintf("%s%s", g.region, g.zones[g.index])
	g.index++
	return zone
}

// OSTypeGenerator generates OS type names
type OSTypeGenerator struct {
	types []string
	index int
}

// NewOSTypeGenerator creates a new OS type generator
func NewOSTypeGenerator() *OSTypeGenerator {
	types := []string{"linux", "darwin", "windows"}
	return &OSTypeGenerator{types: types, index: 0}
}

// Generate generates an OS type
func (g *OSTypeGenerator) Generate() string {
	if g.index >= len(g.types) {
		g.index = 0
	}
	osType := g.types[g.index]
	g.index++
	return osType
}

// GenerateClusterName generates a Kubernetes cluster name
func GenerateClusterName() string {
	environments := []string{"prod", "staging", "dev"}
	regions := []string{"us-east", "us-west", "eu-west", "ap-south"}
	env := RandomChoice(environments)
	region := RandomChoice(regions)
	return fmt.Sprintf("%s-%s-cluster", env, region)
}

// GenerateDeploymentName generates a Kubernetes deployment name
func GenerateDeploymentName(serviceName string) string {
	return fmt.Sprintf("%s-deployment", serviceName)
}
