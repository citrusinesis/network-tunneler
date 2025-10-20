package proxy

import (
	"crypto/tls"

	"go.uber.org/fx"

	"network-tunneler/internal/certs"
	"network-tunneler/pkg/crypto"
	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

const channelBuffer = 100

var Module = fx.Options(
	fx.Provide(
		ProvideConfig,
		ProvideResponseChannel,

		NewPacketForwarder,
		NewServerConnection,

		New,
	),
)

type ProvidedConfig struct {
	fx.Out

	Config       *Config
	TlsConfig    *tls.Config
	LoggerConfig logger.Config
}

func ProvideConfig() (ProvidedConfig, error) {
	cfg := DefaultConfig()

	tlsOpts := cfg.TLS
	if tlsOpts.CertPath == "" && tlsOpts.CertPEM == nil {
		tlsOpts.CertPEM = []byte(certs.ProxyCert)
	}
	if tlsOpts.KeyPath == "" && tlsOpts.KeyPEM == nil {
		tlsOpts.KeyPEM = []byte(certs.ProxyKey)
	}
	if tlsOpts.CAPath == "" && tlsOpts.CAPEM == nil {
		tlsOpts.CAPEM = []byte(certs.CACert)
	}

	tlsConfig, err := crypto.LoadClientTLSConfig(tlsOpts)
	if err != nil {
		return ProvidedConfig{}, err
	}

	return ProvidedConfig{
		Config:       cfg,
		TlsConfig:    tlsConfig,
		LoggerConfig: cfg.Log,
	}, nil
}

func ProvideResponseChannel() chan *pb.Packet {
	return make(chan *pb.Packet, channelBuffer)
}
