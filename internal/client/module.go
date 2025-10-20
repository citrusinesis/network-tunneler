package client

import (
	"crypto/tls"
	"fmt"

	"go.uber.org/fx"

	"network-tunneler/internal/certs"
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
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return ProvidedConfig{}, err
	}

	tlsOpts := cfg.TLS
	if tlsOpts.CertPath == "" && tlsOpts.CertPEM == nil {
		tlsOpts.CertPEM = []byte(certs.ClientCert)
	}
	if tlsOpts.KeyPath == "" && tlsOpts.KeyPEM == nil {
		tlsOpts.KeyPEM = []byte(certs.ClientKey)
	}
	if tlsOpts.CAPath == "" && tlsOpts.CAPEM == nil {
		tlsOpts.CAPEM = []byte(certs.CACert)
	}

	tlsConfig, err := crypto.LoadClientTLSConfig(tlsOpts)
	if err != nil {
		return ProvidedConfig{}, fmt.Errorf("failed to load TLS config: %w", err)
	}

	return ProvidedConfig{
		Config:       cfg,
		TLSConfig:    tlsConfig,
		LoggerConfig: &cfg.Log,
	}, nil
}
