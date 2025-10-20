package server

import (
	"fmt"

	"network-tunneler/internal/config"
	"network-tunneler/pkg/crypto"
	"network-tunneler/pkg/logger"
)

type Config struct {
	ClientListenAddr string            `mapstructure:"client_listen_addr" json:"client_listen_addr" yaml:"client_listen_addr"`
	ProxyListenAddr  string            `mapstructure:"proxy_listen_addr" json:"proxy_listen_addr" yaml:"proxy_listen_addr"`
	TLS              crypto.TLSOptions `mapstructure:"tls" json:"tls" yaml:"tls"`
	Log              logger.Config     `mapstructure:"log" json:"log" yaml:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		ClientListenAddr: ":8080",
		ProxyListenAddr:  ":8081",
		TLS:              crypto.TLSOptions{},
		Log:              config.DefaultLogConfig(),
	}
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	if err := config.Load("server", configFile, cfg); err != nil {
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
