package proxy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
)

type Proxy struct {
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

func New(p Params) (*Proxy, error) {
	proxy := &Proxy{
		config:     p.Config,
		logger:     p.Logger.With(logger.String("component", "proxy")),
		serverConn: p.ServerConn,
		forwarder:  p.Forwarder,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: proxy.start,
		OnStop:  proxy.stop,
	})

	return proxy, nil
}

func (i *Proxy) start(ctx context.Context) error {
	i.logger.Info("starting proxy",
		logger.String("server_addr", i.config.ServerAddr),
		logger.String("proxy_id", i.config.ProxyID),
		logger.String("managed_cidr", i.config.ManagedCIDR),
	)

	if err := i.serverConn.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	go i.heartbeatLoop()

	return nil
}

func (i *Proxy) stop(ctx context.Context) error {
	i.logger.Info("stopping proxy")

	if err := i.serverConn.Close(); err != nil {
		i.logger.Error("failed to close server connection", logger.Error(err))
	}

	i.forwarder.Stop()

	i.logger.Info("proxy stopped")

	return nil
}

func (i *Proxy) heartbeatLoop() {
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
