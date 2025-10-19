package agent

import (
	"fmt"

	"network-tunneler/internal/config"
)

type Config struct {
	ServerAddr string           `mapstructure:"server_addr" json:"server_addr" yaml:"server_addr"`
	ListenPort int              `mapstructure:"listen_port" json:"listen_port" yaml:"listen_port"`
	TargetCIDR string           `mapstructure:"target_cidr" json:"target_cidr" yaml:"target_cidr"`
	TLS        config.TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log        config.LogConfig `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr: "localhost:8080",
		ListenPort: 9999,
		TargetCIDR: "100.64.0.0/10",
		TLS: config.TLSConfig{
			CertPath:           "certs/agent.crt",
			KeyPath:            "certs/agent.key",
			CAPath:             "certs/ca.crt",
			InsecureSkipVerify: false,
		},
		Log: config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile, envFile string) (*Config, error) {
	loader := config.NewLoader("agent")

	if err := loader.LoadEnvFile(envFile); err != nil {
		return nil, fmt.Errorf("failed to load .env: %w", err)
	}

	if err := loader.LoadFile(configFile); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := loader.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerAddr == "" {
		return fmt.Errorf("server address is required")
	}
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
	}
	if c.TargetCIDR == "" {
		return fmt.Errorf("target CIDR is required")
	}
	return nil
}
