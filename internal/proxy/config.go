package proxy

import (
	"fmt"

	"network-tunneler/internal/config"
)

type Config struct {
	ServerAddr  string           `mapstructure:"server_addr" json:"server_addr" yaml:"server_addr"`
	ProxyID   string           `mapstructure:"proxy_id" json:"proxy_id" yaml:"proxy_id"`
	ManagedCIDR string           `mapstructure:"managed_cidr" json:"managed_cidr" yaml:"managed_cidr"`
	TLS         config.TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log         config.LogConfig `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:  "localhost:8081",
		ProxyID:   "proxy-1",
		ManagedCIDR: "192.168.1.0/24",
		TLS:         config.DefaultTLSConfig("proxy"),
		Log:         config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.Load("proxy", configFile, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadConfigMultiple(configFiles ...string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.LoadMultiple("proxy", configFiles, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerAddr == "" {
		return fmt.Errorf("server address is required")
	}
	if c.ProxyID == "" {
		return fmt.Errorf("proxy ID is required")
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
