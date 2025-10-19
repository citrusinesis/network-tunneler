package server

import (
	"fmt"

	"network-tunneler/internal/config"
)

type Config struct {
	AgentListenAddr   string           `mapstructure:"agent_listen_addr" json:"agent_listen_addr" yaml:"agent_listen_addr"`
	ImplantListenAddr string           `mapstructure:"implant_listen_addr" json:"implant_listen_addr" yaml:"implant_listen_addr"`
	TLS               config.TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log               config.LogConfig `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		AgentListenAddr:   ":8080",
		ImplantListenAddr: ":8081",
		TLS: config.TLSConfig{
			CertPath:           "certs/server.crt",
			KeyPath:            "certs/server.key",
			CAPath:             "certs/ca.crt",
			InsecureSkipVerify: false,
		},
		Log: config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile, envFile string) (*Config, error) {
	loader := config.NewLoader("server")

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
	if c.AgentListenAddr == "" {
		return fmt.Errorf("agent listen address is required")
	}
	if c.ImplantListenAddr == "" {
		return fmt.Errorf("implant listen address is required")
	}
	if c.AgentListenAddr == c.ImplantListenAddr {
		return fmt.Errorf("agent and implant listen addresses must be different")
	}
	return nil
}
