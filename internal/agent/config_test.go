package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServerAddr != "localhost:8080" {
		t.Errorf("expected ServerAddr localhost:8080, got %s", cfg.ServerAddr)
	}
	if cfg.ListenPort != 9999 {
		t.Errorf("expected ListenPort 9999, got %d", cfg.ListenPort)
	}
	if cfg.TargetCIDR != "100.64.0.0/10" {
		t.Errorf("expected TargetCIDR 100.64.0.0/10, got %s", cfg.TargetCIDR)
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
				ServerAddr: "",
				ListenPort: 9999,
				TargetCIDR: "100.64.0.0/10",
			},
			expectErr: true,
		},
		{
			name: "invalid port (too low)",
			cfg: &Config{
				ServerAddr: "localhost:8080",
				ListenPort: 0,
				TargetCIDR: "100.64.0.0/10",
			},
			expectErr: true,
		},
		{
			name: "invalid port (too high)",
			cfg: &Config{
				ServerAddr: "localhost:8080",
				ListenPort: 65536,
				TargetCIDR: "100.64.0.0/10",
			},
			expectErr: true,
		},
		{
			name: "missing CIDR",
			cfg: &Config{
				ServerAddr: "localhost:8080",
				ListenPort: 9999,
				TargetCIDR: "",
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

func TestLoadConfig_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
server_addr: "test.example.com:9000"
listen_port: 1234
target_cidr: "10.0.0.0/8"
tls:
  cert_path: "/path/to/cert"
  key_path: "/path/to/key"
  ca_path: "/path/to/ca"
log:
  level: "debug"
  format: "json"
  development: true
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.ServerAddr != "test.example.com:9000" {
		t.Errorf("expected ServerAddr test.example.com:9000, got %s", cfg.ServerAddr)
	}
	if cfg.ListenPort != 1234 {
		t.Errorf("expected ListenPort 1234, got %d", cfg.ListenPort)
	}
	if cfg.TLS.CertPath != "/path/to/cert" {
		t.Errorf("expected CertPath /path/to/cert, got %s", cfg.TLS.CertPath)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "server_addr": "json.example.com:8080",
  "listen_port": 5555,
  "target_cidr": "172.16.0.0/12",
  "tls": {
    "cert_path": "/json/cert",
    "key_path": "/json/key",
    "ca_path": "/json/ca"
  }
}`
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.ServerAddr != "json.example.com:8080" {
		t.Errorf("expected ServerAddr json.example.com:8080, got %s", cfg.ServerAddr)
	}
	if cfg.ListenPort != 5555 {
		t.Errorf("expected ListenPort 5555, got %d", cfg.ListenPort)
	}
}

func TestLoadConfig_DotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	envContent := `AGENT_SERVER_ADDR=env.example.com:7777
AGENT_LISTEN_PORT=3333
AGENT_TARGET_CIDR=192.168.0.0/16
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	cfg, err := LoadConfig(envPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	t.Logf("Loaded ServerAddr: %s", cfg.ServerAddr)
	t.Logf("Loaded ListenPort: %d", cfg.ListenPort)
	t.Logf("ENV AGENT_SERVER_ADDR: %s", os.Getenv("AGENT_SERVER_ADDR"))
	t.Logf("ENV AGENT_LISTEN_PORT: %s", os.Getenv("AGENT_LISTEN_PORT"))

	if cfg.ServerAddr != "env.example.com:7777" {
		t.Errorf("expected ServerAddr from env, got %s", cfg.ServerAddr)
	}
	if cfg.ListenPort != 3333 {
		t.Errorf("expected ListenPort 3333, got %d", cfg.ListenPort)
	}
}

func TestLoadConfigMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
server_addr: "yaml.example.com:8080"
listen_port: 1111
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	envPath := filepath.Join(tmpDir, ".env")
	envContent := `AGENT_LISTEN_PORT=2222
AGENT_TARGET_CIDR=10.0.0.0/8
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	cfg, err := LoadConfigMultiple(yamlPath, envPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.ServerAddr != "yaml.example.com:8080" {
		t.Errorf("expected ServerAddr from yaml, got %s", cfg.ServerAddr)
	}
	if cfg.ListenPort != 2222 {
		t.Errorf("expected ListenPort 2222 from env (override), got %d", cfg.ListenPort)
	}
	if cfg.TargetCIDR != "10.0.0.0/8" {
		t.Errorf("expected TargetCIDR from env, got %s", cfg.TargetCIDR)
	}
}
