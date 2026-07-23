package config

import "testing"

func baseTracesCfg() *GeneratorConfig {
	return &GeneratorConfig{
		Output: OutputConfig{Directory: "/tmp/out"},
		Traces: TracesConfig{
			Count:    10,
			Spans:    SpansConfig{AvgPerTrace: 5, StdDev: 1},
			Services: ServicesConfig{Count: 1, Names: []string{"api"}},
		},
	}
}

func TestValidateCustomAttributes(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *GeneratorConfig)
		wantErr bool
	}{
		{"legacy only ok", func(c *GeneratorConfig) { c.Traces.CustomAttributes.Count = 30 }, false},
		{"fat ok", func(c *GeneratorConfig) {
			c.Traces.CustomAttributes.PerSpanMin = 5
			c.Traces.CustomAttributes.PerSpanMax = 10
			c.Traces.CustomAttributes.ValueBytes = 1000
		}, false},
		{"max < min", func(c *GeneratorConfig) {
			c.Traces.CustomAttributes.PerSpanMin = 10
			c.Traces.CustomAttributes.PerSpanMax = 5
			c.Traces.CustomAttributes.ValueBytes = 100
		}, true},
		{"fat without value_bytes", func(c *GeneratorConfig) {
			c.Traces.CustomAttributes.PerSpanMax = 5
		}, true},
		{"negative value_bytes", func(c *GeneratorConfig) {
			c.Traces.CustomAttributes.ValueBytes = -1
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := baseTracesCfg()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRootConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *GeneratorConfig)
		wantErr bool
	}{
		{"rootless ok", func(c *GeneratorConfig) {
			c.Traces.Root.Rootless = RootlessConfig{Enabled: true, Percentage: 50}
		}, false},
		{"percentage too high", func(c *GeneratorConfig) {
			c.Traces.Root.Rootless = RootlessConfig{Enabled: true, Percentage: 150}
		}, true},
		{"late root ok", func(c *GeneratorConfig) {
			c.Traces.Root.LateRoot = LateRootConfig{Enabled: true, Percentage: 100, DelayMs: 90000}
		}, false},
		{"late root without delay", func(c *GeneratorConfig) {
			c.Traces.Root.LateRoot = LateRootConfig{Enabled: true, Percentage: 100}
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := baseTracesCfg()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestEstimateMemoryFatSpans(t *testing.T) {
	c := baseTracesCfg()
	c.Traces.Count = 100
	c.Traces.Spans.AvgPerTrace = 10 // 1000 spans

	base := c.EstimateMemoryUsage()
	wantBase := int64(1000) * bytesPerSpan
	if base != wantBase {
		t.Fatalf("base estimate = %d, want %d", base, wantBase)
	}

	c.Traces.CustomAttributes.PerSpanMin = 20
	c.Traces.CustomAttributes.PerSpanMax = 20
	c.Traces.CustomAttributes.ValueBytes = 1500
	fat := c.EstimateMemoryUsage()
	perSpan := int64(bytesPerSpan) + 20*(1500+fatAttrKeyOverhead)
	if want := int64(1000) * perSpan; fat != want {
		t.Fatalf("fat estimate = %d, want %d", fat, want)
	}
	if fat <= base {
		t.Fatalf("fat estimate (%d) should exceed base (%d)", fat, base)
	}
}

func TestMemoryCapAndOverride(t *testing.T) {
	c := baseTracesCfg()
	// Huge fat config that blows past the default 10GB cap.
	c.Traces.Count = 100000
	c.Traces.Spans.AvgPerTrace = 50
	c.Traces.CustomAttributes.PerSpanMin = 20
	c.Traces.CustomAttributes.PerSpanMax = 20
	c.Traces.CustomAttributes.ValueBytes = 2000

	if err := c.Validate(); err == nil {
		t.Fatal("expected memory cap to trip, got nil error")
	}

	// Raising the cap should allow it.
	c.Limits.MaxMemoryGB = 100000
	if err := c.Validate(); err != nil {
		t.Fatalf("raised cap should pass, got %v", err)
	}

	// allow_unbounded should also bypass the check.
	c.Limits.MaxMemoryGB = 0
	c.Limits.AllowUnbounded = true
	if err := c.Validate(); err != nil {
		t.Fatalf("allow_unbounded should pass, got %v", err)
	}
}

func TestApplyDefaultsNewFields(t *testing.T) {
	c := baseTracesCfg()
	c.Traces.CustomAttributes.PerSpanMax = 5
	c.Traces.CustomAttributes.ValueBytes = 100
	c.ApplyDefaults()

	if c.Limits.MaxMemoryGB != defaultMaxMemoryGB {
		t.Errorf("MaxMemoryGB default = %v, want %v", c.Limits.MaxMemoryGB, float64(defaultMaxMemoryGB))
	}
	if c.Traces.CustomAttributes.KeyPrefix != "custom.fat" {
		t.Errorf("KeyPrefix default = %q, want custom.fat", c.Traces.CustomAttributes.KeyPrefix)
	}
}
