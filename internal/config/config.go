package config

import (
	"network-tunneler/pkg/logger"
)

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
