package server

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ClientListenAddr != ":8080" {
		t.Errorf("expected ClientListenAddr :8080, got %s", cfg.ClientListenAddr)
	}
	if cfg.ProxyListenAddr != ":8081" {
		t.Errorf("expected ProxyListenAddr :8081, got %s", cfg.ProxyListenAddr)
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
			name: "missing client listen addr",
			cfg: &Config{
				ClientListenAddr:   "",
				ProxyListenAddr: ":8081",
			},
			expectErr: true,
		},
		{
			name: "missing proxy listen addr",
			cfg: &Config{
				ClientListenAddr:   ":8080",
				ProxyListenAddr: "",
			},
			expectErr: true,
		},
		{
			name: "same addresses",
			cfg: &Config{
				ClientListenAddr:   ":8080",
				ProxyListenAddr: ":8080",
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
