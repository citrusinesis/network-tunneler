package server

import (
	"fmt"

	"network-tunneler/internal/config"
)

type Config struct {
	ClientListenAddr   string           `mapstructure:"client_listen_addr" json:"client_listen_addr" yaml:"client_listen_addr"`
	ProxyListenAddr string           `mapstructure:"proxy_listen_addr" json:"proxy_listen_addr" yaml:"proxy_listen_addr"`
	TLS               config.TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log               config.LogConfig `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		ClientListenAddr:   ":8080",
		ProxyListenAddr: ":8081",
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
	if c.ClientListenAddr == "" {
		return fmt.Errorf("client listen address is required")
	}
	if c.ProxyListenAddr == "" {
		return fmt.Errorf("proxy listen address is required")
	}
	if c.ClientListenAddr == c.ProxyListenAddr {
		return fmt.Errorf("client and proxy listen addresses must be different")
	}
	return nil
}

func (c *Config) GetTLS() *config.TLSConfig {
	return &c.TLS
}

func (c *Config) GetLog() *config.LogConfig {
	return &c.Log
}
