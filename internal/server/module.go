package server

import (
	"crypto/tls"

	"go.uber.org/fx"

	"network-tunneler/internal/certs"
	"network-tunneler/pkg/crypto"
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
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return ProvidedConfig{}, err
	}

	tlsOpts := cfg.TLS
	if tlsOpts.CertPath == "" && tlsOpts.CertPEM == nil {
		tlsOpts.CertPEM = []byte(certs.ServerCert)
	}
	if tlsOpts.KeyPath == "" && tlsOpts.KeyPEM == nil {
		tlsOpts.KeyPEM = []byte(certs.ServerKey)
	}
	if tlsOpts.CAPath == "" && tlsOpts.CAPEM == nil {
		tlsOpts.CAPEM = []byte(certs.CACert)
	}

	tlsConfig, err := crypto.LoadServerTLSConfig(tlsOpts)
	if err != nil {
		return ProvidedConfig{}, err
	}

	return ProvidedConfig{
		Config:       cfg,
		TLSConfig:    tlsConfig,
		LoggerConfig: &cfg.Log,
	}, nil
}
