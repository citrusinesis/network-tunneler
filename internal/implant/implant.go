package implant

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
)

type Implant struct {
	config     *Config
	logger     logger.Logger
	serverConn *ServerConnection
	forwarder  *PacketForwarder
}

type Params struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Config     *Config
	Logger     logger.Logger
	ServerConn *ServerConnection
	Forwarder  *PacketForwarder
}

func New(p Params) (*Implant, error) {
	implant := &Implant{
		config:     p.Config,
		logger:     p.Logger.With(logger.String("component", "implant")),
		serverConn: p.ServerConn,
		forwarder:  p.Forwarder,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: implant.start,
		OnStop:  implant.stop,
	})

	return implant, nil
}

func (i *Implant) start(ctx context.Context) error {
	i.logger.Info("starting implant",
		logger.String("server_addr", i.config.ServerAddr),
		logger.String("implant_id", i.config.ImplantID),
		logger.String("managed_cidr", i.config.ManagedCIDR),
	)

	if err := i.serverConn.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	go i.heartbeatLoop()

	return nil
}

func (i *Implant) stop(ctx context.Context) error {
	i.logger.Info("stopping implant")

	if err := i.serverConn.Close(); err != nil {
		i.logger.Error("failed to close server connection", logger.Error(err))
	}

	i.forwarder.Stop()

	i.logger.Info("implant stopped")

	return nil
}

func (i *Implant) heartbeatLoop() {
	defer i.logger.Info("heartbeat loop stopped")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := i.serverConn.SendHeartbeat(); err != nil {
				i.logger.Error("failed to send heartbeat", logger.Error(err))
			}
		case <-i.serverConn.stopChan:
			return
		}
	}
}
