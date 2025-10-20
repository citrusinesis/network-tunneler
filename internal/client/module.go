package client

import (
	"crypto/tls"
	"fmt"

	"go.uber.org/fx"

	"network-tunneler/pkg/crypto"
	"network-tunneler/pkg/logger"
)

var Module = fx.Module("client",
	fx.Provide(
		ProvideConfig,

		NewConnectionTracker,
		NewNetfilterManager,
		NewServerConnection,

		New,
	),
)

type ProvidedConfig struct {
	fx.Out

	Config       *Config
	TLSConfig    *tls.Config
	LoggerConfig *logger.Config
}

func ProvideConfig(configFile string) (ProvidedConfig, error) {
	var cfg *Config
	var err error

	if configFile == "" {
		cfg = DefaultConfig()
	} else {
		cfg, err = LoadConfig(configFile)
		if err != nil {
			return ProvidedConfig{}, err
		}
	}

	tlsConfig, err := crypto.LoadClientTLSConfig(cfg.GetTLS())
	if err != nil {
		return ProvidedConfig{}, fmt.Errorf("failed to load TLS config: %w", err)
	}

	return ProvidedConfig{
		Config:       cfg,
		TLSConfig:    tlsConfig,
		LoggerConfig: cfg.Log.ToLoggerConfig(),
	}, nil
}
