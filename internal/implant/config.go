package implant

import (
	"fmt"

	"network-tunneler/internal/config"
)

type Config struct {
	ServerAddr  string           `mapstructure:"server_addr" json:"server_addr" yaml:"server_addr"`
	ImplantID   string           `mapstructure:"implant_id" json:"implant_id" yaml:"implant_id"`
	ManagedCIDR string           `mapstructure:"managed_cidr" json:"managed_cidr" yaml:"managed_cidr"`
	TLS         config.TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log         config.LogConfig `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:  "localhost:8081",
		ImplantID:   "implant-1",
		ManagedCIDR: "192.168.1.0/24",
		TLS: config.TLSConfig{
			CertPath:           "certs/implant.crt",
			KeyPath:            "certs/implant.key",
			CAPath:             "certs/ca.crt",
			InsecureSkipVerify: false,
		},
		Log: config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile, envFile string) (*Config, error) {
	loader := config.NewLoader("implant")

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
	if c.ImplantID == "" {
		return fmt.Errorf("implant ID is required")
	}
	if c.ManagedCIDR == "" {
		return fmt.Errorf("managed CIDR is required")
	}
	return nil
}
