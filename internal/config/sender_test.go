package config

import "testing"

func baseSenderCfg() *SenderConfig {
	return &SenderConfig{
		Input: InputConfig{Traces: "/tmp/t.pb"},
		OTLP:  OTLPConfig{Endpoint: "localhost:4317"},
		Sending: SendingConfig{
			RateLimit:   RateLimitConfig{EventsPerSecond: 1000},
			Concurrency: 4,
		},
	}
}

func TestSenderDeferredDefaults(t *testing.T) {
	c := baseSenderCfg()
	c.ApplyDefaults()

	if c.Sending.Deferred.DrainTimeout != "120s" {
		t.Errorf("drain_timeout default = %q, want 120s", c.Sending.Deferred.DrainTimeout)
	}
	if c.Sending.Deferred.MaxPending != 100000 {
		t.Errorf("max_pending default = %d, want 100000", c.Sending.Deferred.MaxPending)
	}

	d, err := c.GetDeferredDrainTimeout()
	if err != nil {
		t.Fatalf("GetDeferredDrainTimeout error: %v", err)
	}
	if d.Seconds() != 120 {
		t.Errorf("parsed drain timeout = %v, want 120s", d)
	}
}

func TestSenderDeferredValidation(t *testing.T) {
	c := baseSenderCfg()
	c.Sending.Deferred.DrainTimeout = "not-a-duration"
	if err := c.Validate(); err == nil {
		t.Error("expected error for bad drain_timeout")
	}

	c = baseSenderCfg()
	c.Sending.Deferred.MaxPending = -1
	if err := c.Validate(); err == nil {
		t.Error("expected error for negative max_pending")
	}

	c = baseSenderCfg()
	c.Sending.Deferred.DrainTimeout = "90s"
	if err := c.Validate(); err != nil {
		t.Errorf("valid deferred config rejected: %v", err)
	}
}
