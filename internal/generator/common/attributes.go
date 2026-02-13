package common

import (
	"fmt"

	"go.opentelemetry.io/proto/otlp/common/v1"
)

// AttributeType represents the type of an attribute
type AttributeType int

const (
	AttributeTypeString AttributeType = iota
	AttributeTypeInt
	AttributeTypeFloat
	AttributeTypeBool
)

// AttributeSchema defines a custom attribute schema
type AttributeSchema struct {
	Name string
	Type AttributeType
}

// GenerateCustomAttributeSchemas generates a set of custom attribute schemas
func GenerateCustomAttributeSchemas(count int) []AttributeSchema {
	schemas := make([]AttributeSchema, count)
	types := []AttributeType{
		AttributeTypeString,
		AttributeTypeInt,
		AttributeTypeFloat,
		AttributeTypeBool,
	}

	for i := 0; i < count; i++ {
		schemas[i] = AttributeSchema{
			Name: fmt.Sprintf("custom.attr.%d", i+1),
			Type: types[i%len(types)], // Distribute types evenly
		}
	}

	return schemas
}

// CreateAttribute creates an OTLP attribute with a random value based on schema
func CreateAttribute(schema AttributeSchema) *v1.KeyValue {
	kv := &v1.KeyValue{
		Key: schema.Name,
	}

	switch schema.Type {
	case AttributeTypeString:
		values := []string{"low", "medium", "high", "critical", "alpha", "beta", "gamma"}
		kv.Value = &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: RandomChoice(values),
			},
		}
	case AttributeTypeInt:
		kv.Value = &v1.AnyValue{
			Value: &v1.AnyValue_IntValue{
				IntValue: int64(RandomInt(1, 1000)),
			},
		}
	case AttributeTypeFloat:
		kv.Value = &v1.AnyValue{
			Value: &v1.AnyValue_DoubleValue{
				DoubleValue: RandomFloat64(0.0, 100.0),
			},
		}
	case AttributeTypeBool:
		kv.Value = &v1.AnyValue{
			Value: &v1.AnyValue_BoolValue{
				BoolValue: RandomBool(),
			},
		}
	}

	return kv
}

// CreateStringAttribute creates a string attribute
func CreateStringAttribute(key, value string) *v1.KeyValue {
	return &v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: value,
			},
		},
	}
}

// CreateIntAttribute creates an integer attribute
func CreateIntAttribute(key string, value int64) *v1.KeyValue {
	return &v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_IntValue{
				IntValue: value,
			},
		},
	}
}

// CreateFloatAttribute creates a float attribute
func CreateFloatAttribute(key string, value float64) *v1.KeyValue {
	return &v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
}

// CreateBoolAttribute creates a boolean attribute
func CreateBoolAttribute(key string, value bool) *v1.KeyValue {
	return &v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_BoolValue{
				BoolValue: value,
			},
		},
	}
}

// CreateHTTPAttributes creates HTTP semantic convention attributes
func CreateHTTPAttributes(method, path string, statusCode int) []*v1.KeyValue {
	attrs := []*v1.KeyValue{
		CreateStringAttribute("http.method", method),
		CreateStringAttribute("http.target", path),
		CreateIntAttribute("http.status_code", int64(statusCode)),
	}

	// Add optional attributes randomly
	if RandomBool() {
		attrs = append(attrs, CreateStringAttribute("http.user_agent", "Mozilla/5.0"))
	}
	if RandomBool() {
		attrs = append(attrs, CreateIntAttribute("http.response_content_length", RandomInt64(100, 50000)))
	}

	return attrs
}

// CreateDBAttributes creates database semantic convention attributes
func CreateDBAttributes(system, statement string) []*v1.KeyValue {
	attrs := []*v1.KeyValue{
		CreateStringAttribute("db.system", system),
		CreateStringAttribute("db.statement", statement),
	}

	// Add optional attributes
	if system == "postgresql" || system == "mysql" {
		attrs = append(attrs, CreateStringAttribute("db.name", "production"))
		attrs = append(attrs, CreateStringAttribute("db.user", "app_user"))
	}

	return attrs
}

// CreateServiceAttributes creates service resource attributes
func CreateServiceAttributes(serviceName string) []*v1.KeyValue {
	return []*v1.KeyValue{
		CreateStringAttribute("service.name", serviceName),
		CreateStringAttribute("service.version", fmt.Sprintf("1.%d.%d", RandomInt(0, 10), RandomInt(0, 20))),
	}
}

// CreateHostAttributes creates host resource attributes
func CreateHostAttributes(hostname, osType string) []*v1.KeyValue {
	return []*v1.KeyValue{
		CreateStringAttribute("host.name", hostname),
		CreateStringAttribute("os.type", osType),
	}
}

// CreateK8sAttributes creates Kubernetes resource attributes
func CreateK8sAttributes(clusterName, namespace, podName, containerName, nodeName string) []*v1.KeyValue {
	attrs := []*v1.KeyValue{
		CreateStringAttribute("k8s.cluster.name", clusterName),
		CreateStringAttribute("k8s.namespace.name", namespace),
	}

	if podName != "" {
		attrs = append(attrs, CreateStringAttribute("k8s.pod.name", podName))
	}
	if containerName != "" {
		attrs = append(attrs, CreateStringAttribute("container.name", containerName))
	}
	if nodeName != "" {
		attrs = append(attrs, CreateStringAttribute("k8s.node.name", nodeName))
	}

	return attrs
}

// CreateCloudAttributes creates cloud resource attributes
func CreateCloudAttributes(provider, region, zone string) []*v1.KeyValue {
	return []*v1.KeyValue{
		CreateStringAttribute("cloud.provider", provider),
		CreateStringAttribute("cloud.region", region),
		CreateStringAttribute("cloud.availability_zone", zone),
	}
}
