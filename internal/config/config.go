package config

import (
	"fmt"

	"network-tunneler/pkg/logger"
)

type Config interface {
	Validate() error
	GetTLS() *TLSConfig
	GetLog() *LogConfig
}

type TLSConfig struct {
	CertPath           string `mapstructure:"cert_path" json:"cert_path" yaml:"cert_path"`
	KeyPath            string `mapstructure:"key_path" json:"key_path" yaml:"key_path"`
	CAPath             string `mapstructure:"ca_path" json:"ca_path" yaml:"ca_path"`
	ServerName         string `mapstructure:"server_name" json:"server_name" yaml:"server_name"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

type LogConfig struct {
	Level       logger.Level  `mapstructure:"level" json:"level" yaml:"level"`
	Format      logger.Format `mapstructure:"format" json:"format" yaml:"format"`
	Development bool          `mapstructure:"development" json:"development" yaml:"development"`
}

func (c *LogConfig) ToLoggerConfig() *logger.Config {
	return &logger.Config{
		Level:       c.Level,
		Format:      c.Format,
		Development: c.Development,
	}
}

func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:       logger.LevelInfo,
		Format:      logger.FormatJSON,
		Development: false,
	}
}

func DevelopmentLogConfig() LogConfig {
	return LogConfig{
		Level:       logger.LevelDebug,
		Format:      logger.FormatConsole,
		Development: true,
	}
}

func DefaultTLSConfig(component string) TLSConfig {
	return TLSConfig{
		CertPath:           "", // Empty = use embedded certificates
		KeyPath:            "", // Empty = use embedded keys
		CAPath:             "", // Empty = use embedded CA
		InsecureSkipVerify: false,
	}
}

func Load(appName string, configFile string, cfg Config) error {
	loader := NewLoader(appName)

	if err := loader.LoadFile(configFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := loader.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

func LoadMultiple(appName string, configFiles []string, cfg Config) error {
	loader := NewLoader(appName)

	for _, file := range configFiles {
		if err := loader.LoadFile(file); err != nil {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	if err := loader.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}
