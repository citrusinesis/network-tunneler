package server

import (
	"crypto/tls"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
)

var Module = fx.Module("server",
	fx.Provide(
		ProvideConfig,

		NewRegistry,
		NewGRPCServer,

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

	tlsConfig, err := LoadTLSConfig(cfg.GetTLS())
	if err != nil {
		return ProvidedConfig{}, err
	}

	return ProvidedConfig{
		Config:       cfg,
		TLSConfig:    tlsConfig,
		LoggerConfig: cfg.Log.ToLoggerConfig(),
	}, nil
}
