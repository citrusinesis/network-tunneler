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
		TLS:        config.DefaultTLSConfig("agent"),
		Log:        config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.Load("agent", configFile, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadConfigMultiple(configFiles ...string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.LoadMultiple("agent", configFiles, cfg); err != nil {
		return nil, err
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

func (c *Config) GetTLS() *config.TLSConfig {
	return &c.TLS
}

func (c *Config) GetLog() *config.LogConfig {
	return &c.Log
}
