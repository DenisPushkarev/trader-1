package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("NATS_URL", "")
	t.Setenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxSignalsPerMinute != 0 {
		t.Errorf("expected MaxSignalsPerMinute=0, got %d", cfg.MaxSignalsPerMinute)
	}
	if cfg.NATSUrl != "nats://localhost:4222" {
		t.Errorf("expected default NATSUrl, got %q", cfg.NATSUrl)
	}
}

func TestLoad_MaxSignalsPerMinute(t *testing.T) {
	t.Setenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE", "100")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxSignalsPerMinute != 100 {
		t.Errorf("expected MaxSignalsPerMinute=100, got %d", cfg.MaxSignalsPerMinute)
	}
}

func TestLoad_MaxSignalsPerMinute_Zero(t *testing.T) {
	t.Setenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxSignalsPerMinute != 0 {
		t.Errorf("expected MaxSignalsPerMinute=0, got %d", cfg.MaxSignalsPerMinute)
	}
}

func TestLoad_MaxSignalsPerMinute_Negative(t *testing.T) {
	t.Setenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative MaxSignalsPerMinute, got nil")
	}
}

func TestLoad_MaxSignalsPerMinute_Invalid(t *testing.T) {
	t.Setenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MaxSignalsPerMinute, got nil")
	}
}

func TestValidate_Negative(t *testing.T) {
	cfg := &Config{MaxSignalsPerMinute: -5}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for negative value")
	}
}

func TestValidate_Zero(t *testing.T) {
	cfg := &Config{MaxSignalsPerMinute: 0}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error for zero value: %v", err)
	}
}

func TestValidate_Positive(t *testing.T) {
	cfg := &Config{MaxSignalsPerMinute: 60}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error for positive value: %v", err)
	}
}
