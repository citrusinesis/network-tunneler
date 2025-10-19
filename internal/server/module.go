package server

import (
	"go.uber.org/fx"

	"network-tunneler/internal/config"
	"network-tunneler/internal/server/handler"
	"network-tunneler/pkg/logger"
)

var Module = fx.Options(
	fx.Provide(
		ProvideConfig,
		ProvideTLSConfig,

		NewTLSManager,
		NewListenerManager,
		NewRegistry,
		NewRouter,

		NewAgentHandler,
		NewImplantHandler,

		New,
	),
	fx.Provide(func(cfg *Config) *logger.Config {
		return cfg.Log.ToLoggerConfig()
	}),
	fx.Invoke(func(*Server) {}),
)

func ProvideConfig(configFile string) (*Config, error) {
	if configFile == "" {
		return DefaultConfig(), nil
	}
	return LoadConfig(configFile)
}

func ProvideTLSConfig(cfg *Config) *config.TLSConfig {
	return &cfg.TLS
}

func NewAgentHandler(registry *Registry, router *Router, logger logger.Logger) *handler.Agent {
	return handler.NewAgent(registry, router, logger)
}

func NewImplantHandler(registry *Registry, router *Router, logger logger.Logger) *handler.Implant {
	return handler.NewImplant(registry, router, logger)
}
