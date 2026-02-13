package traces

import (
	"github.com/honeycomb/telemetry-gen-and-send/internal/generator/common"
)

// OperationType represents the type of operation a span performs
type OperationType int

const (
	OperationTypeHTTP OperationType = iota
	OperationTypeDB
	OperationTypeInternal
)

// ServiceNode represents a service in the topology
type ServiceNode struct {
	Name          string
	IsIngress     bool
	Operations    []Operation
	Downstream    []*ServiceNode
}

// Operation represents an operation that a service can perform
type Operation struct {
	Name string
	Type OperationType

	// HTTP specific
	HTTPMethod string
	HTTPPath   string

	// DB specific
	DBSystem    string
	DBStatement string
}

// ServiceTopology represents the overall service graph
type ServiceTopology struct {
	Services       []*ServiceNode
	IngressServices []*ServiceNode
}

// BuildTopology builds a service topology from service names and configuration
func BuildTopology(serviceNames []string, singleIngress bool, ingressService string) *ServiceTopology {
	topology := &ServiceTopology{
		Services:       make([]*ServiceNode, 0, len(serviceNames)),
		IngressServices: make([]*ServiceNode, 0),
	}

	// Create all service nodes
	serviceMap := make(map[string]*ServiceNode)
	for _, name := range serviceNames {
		node := &ServiceNode{
			Name:       name,
			IsIngress:  false,
			Operations: generateOperationsForService(name),
			Downstream: make([]*ServiceNode, 0),
		}
		serviceMap[name] = node
		topology.Services = append(topology.Services, node)
	}

	// Set ingress services
	if singleIngress {
		if ingress, ok := serviceMap[ingressService]; ok {
			ingress.IsIngress = true
			topology.IngressServices = append(topology.IngressServices, ingress)
		}
	} else {
		// Multiple ingresses - first 1-2 services
		count := 1
		if len(serviceNames) > 2 {
			count = 2
		}
		for i := 0; i < count && i < len(topology.Services); i++ {
			topology.Services[i].IsIngress = true
			topology.IngressServices = append(topology.IngressServices, topology.Services[i])
		}
	}

	// Build downstream relationships
	// Simple approach: each service can call services after it in the list
	for i, service := range topology.Services {
		if i < len(topology.Services)-1 {
			// Can call 1-3 downstream services
			downstreamCount := common.RandomInt(1, 3)
			if i+downstreamCount > len(topology.Services) {
				downstreamCount = len(topology.Services) - i - 1
			}

			for j := 1; j <= downstreamCount && i+j < len(topology.Services); j++ {
				service.Downstream = append(service.Downstream, topology.Services[i+j])
			}
		}
	}

	return topology
}

// generateOperationsForService generates a set of operations for a service
func generateOperationsForService(serviceName string) []Operation {
	operations := make([]Operation, 0)

	// Every service has some HTTP operations
	httpOps := common.RandomInt(2, 5)
	for i := 0; i < httpOps; i++ {
		operations = append(operations, Operation{
			Name:       common.RandomHTTPPath(),
			Type:       OperationTypeHTTP,
			HTTPMethod: common.RandomHTTPMethod(),
			HTTPPath:   common.RandomHTTPPath(),
		})
	}

	// Most services have DB operations
	if common.RandomBool() {
		dbOps := common.RandomInt(1, 3)
		dbSystem := common.RandomDBSystem()
		for i := 0; i < dbOps; i++ {
			operations = append(operations, Operation{
				Name:        "db.query",
				Type:        OperationTypeDB,
				DBSystem:    dbSystem,
				DBStatement: common.RandomDBStatement(dbSystem),
			})
		}
	}

	// Some internal operations
	internalOps := common.RandomInt(1, 2)
	for i := 0; i < internalOps; i++ {
		operations = append(operations, Operation{
			Name: "internal.process",
			Type: OperationTypeInternal,
		})
	}

	return operations
}

// GetRandomIngress returns a random ingress service
func (t *ServiceTopology) GetRandomIngress() *ServiceNode {
	if len(t.IngressServices) == 0 {
		return nil
	}
	return common.RandomChoice(t.IngressServices)
}

// GetRandomOperation returns a random operation from a service
func (s *ServiceNode) GetRandomOperation() Operation {
	if len(s.Operations) == 0 {
		return Operation{
			Name: "unknown",
			Type: OperationTypeInternal,
		}
	}
	return common.RandomChoice(s.Operations)
}

// GetRandomDownstream returns a random downstream service, or nil
func (s *ServiceNode) GetRandomDownstream() *ServiceNode {
	if len(s.Downstream) == 0 {
		return nil
	}
	return common.RandomChoice(s.Downstream)
}

// HasDownstream returns true if the service has downstream services
func (s *ServiceNode) HasDownstream() bool {
	return len(s.Downstream) > 0
}
