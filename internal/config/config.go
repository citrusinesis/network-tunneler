package config

import (
	"fmt"

	"network-tunneler/internal/version"
	"network-tunneler/pkg/logger"
)

func DefaultLogConfig() logger.Config {
	if version.IsDebug() {
		return DevelopmentLogConfig()
	}
	return logger.Config{
		Level:       logger.LevelInfo,
		Format:      logger.FormatJSON,
		Development: false,
	}
}

func DevelopmentLogConfig() logger.Config {
	return logger.Config{
		Level:       logger.LevelDebug,
		Format:      logger.FormatConsole,
		Development: true,
	}
}

type Validator interface {
	Validate() error
}

func Load[T Validator](appName string, configFile string, cfg T) error {
	if configFile == "" {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		return nil
	}

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
