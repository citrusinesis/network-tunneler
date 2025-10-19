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
		TLS:         config.DefaultTLSConfig("implant"),
		Log:         config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.Load("implant", configFile, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadConfigMultiple(configFiles ...string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.LoadMultiple("implant", configFiles, cfg); err != nil {
		return nil, err
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

func (c *Config) GetTLS() *config.TLSConfig {
	return &c.TLS
}

func (c *Config) GetLog() *config.LogConfig {
	return &c.Log
}
