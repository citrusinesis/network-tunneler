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
		TLS:               config.DefaultTLSConfig("server"),
		Log:               config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.Load("server", configFile, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadConfigMultiple(configFiles ...string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.LoadMultiple("server", configFiles, cfg); err != nil {
		return nil, err
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

func (c *Config) GetTLS() *config.TLSConfig {
	return &c.TLS
}

func (c *Config) GetLog() *config.LogConfig {
	return &c.Log
}
