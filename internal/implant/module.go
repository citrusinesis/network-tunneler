package implant

import (
	"crypto/tls"

	"go.uber.org/fx"

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

func ProvideConfig(log logger.Logger) (ProvidedConfig, error) {
	cfg := DefaultConfig()

	tlsConfig, err := LoadTLSConfig(cfg.GetTLS(), log)
	if err != nil {
		return ProvidedConfig{}, err
	}

	return ProvidedConfig{
		Config:       cfg,
		TlsConfig:    tlsConfig,
		LoggerConfig: *cfg.Log.ToLoggerConfig(),
	}, nil
}

func ProvideResponseChannel() chan *pb.Packet {
	return make(chan *pb.Packet, channelBuffer)
}
