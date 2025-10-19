package server

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AgentListenAddr != ":8080" {
		t.Errorf("expected AgentListenAddr :8080, got %s", cfg.AgentListenAddr)
	}
	if cfg.ImplantListenAddr != ":8081" {
		t.Errorf("expected ImplantListenAddr :8081, got %s", cfg.ImplantListenAddr)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		expectErr bool
	}{
		{
			name:      "valid config",
			cfg:       DefaultConfig(),
			expectErr: false,
		},
		{
			name: "missing agent listen addr",
			cfg: &Config{
				AgentListenAddr:   "",
				ImplantListenAddr: ":8081",
			},
			expectErr: true,
		},
		{
			name: "missing implant listen addr",
			cfg: &Config{
				AgentListenAddr:   ":8080",
				ImplantListenAddr: "",
			},
			expectErr: true,
		},
		{
			name: "same addresses",
			cfg: &Config{
				AgentListenAddr:   ":8080",
				ImplantListenAddr: ":8080",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
