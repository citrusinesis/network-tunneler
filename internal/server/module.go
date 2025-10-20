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
	TlsConfig    *tls.Config
	LoggerConfig *logger.Config
}

func ProvideConfig(configFile string, log logger.Logger) (ProvidedConfig, error) {
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

	tlsConfig, err := LoadTLSConfig(cfg.GetTLS(), log)
	if err != nil {
		return ProvidedConfig{}, err
	}

	return ProvidedConfig{
		Config:       cfg,
		TlsConfig:    tlsConfig,
		LoggerConfig: cfg.Log.ToLoggerConfig(),
	}, nil
}
