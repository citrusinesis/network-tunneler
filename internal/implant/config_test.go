package implant

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServerAddr != "localhost:8081" {
		t.Errorf("expected ServerAddr localhost:8081, got %s", cfg.ServerAddr)
	}
	if cfg.ImplantID != "implant-1" {
		t.Errorf("expected ImplantID implant-1, got %s", cfg.ImplantID)
	}
	if cfg.ManagedCIDR != "192.168.1.0/24" {
		t.Errorf("expected ManagedCIDR 192.168.1.0/24, got %s", cfg.ManagedCIDR)
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
			name: "missing server addr",
			cfg: &Config{
				ServerAddr:  "",
				ImplantID:   "implant-1",
				ManagedCIDR: "192.168.1.0/24",
			},
			expectErr: true,
		},
		{
			name: "missing implant ID",
			cfg: &Config{
				ServerAddr:  "localhost:8081",
				ImplantID:   "",
				ManagedCIDR: "192.168.1.0/24",
			},
			expectErr: true,
		},
		{
			name: "missing managed CIDR",
			cfg: &Config{
				ServerAddr:  "localhost:8081",
				ImplantID:   "implant-1",
				ManagedCIDR: "",
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
