package agent

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
)

type Agent struct {
	config     *Config
	logger     logger.Logger
	netfilter  *NetfilterManager
	tracker    *ConnectionTracker
	serverConn *ServerConnection
	listener   net.Listener
}

type Params struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Config     *Config
	Logger     logger.Logger
	Netfilter  *NetfilterManager
	Tracker    *ConnectionTracker
	ServerConn *ServerConnection
}

func New(p Params) (*Agent, error) {
	agent := &Agent{
		config:     p.Config,
		logger:     p.Logger.With(logger.String("component", "agent")),
		netfilter:  p.Netfilter,
		tracker:    p.Tracker,
		serverConn: p.ServerConn,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: agent.start,
		OnStop:  agent.stop,
	})

	return agent, nil
}

func (a *Agent) start(ctx context.Context) error {
	a.logger.Info("starting agent",
		logger.String("server_addr", a.config.ServerAddr),
		logger.Int("listen_port", a.config.ListenPort),
		logger.String("target_cidr", a.config.TargetCIDR),
	)

	if err := a.serverConn.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := a.netfilter.Setup(); err != nil {
		return fmt.Errorf("failed to setup netfilter: %w", err)
	}

	listenAddr := fmt.Sprintf(":%d", a.config.ListenPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		a.netfilter.Cleanup()
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	a.listener = listener
	a.logger.Info("listening for redirected connections",
		logger.String("listen_addr", listenAddr),
	)

	go a.acceptLoop()
	go a.cleanupLoop()

	return nil
}

func (a *Agent) stop(ctx context.Context) error {
	a.logger.Info("stopping agent")

	if a.listener != nil {
		a.listener.Close()
	}

	if err := a.netfilter.Cleanup(); err != nil {
		a.logger.Error("failed to cleanup netfilter", logger.Error(err))
	}

	if err := a.serverConn.Close(); err != nil {
		a.logger.Error("failed to close server connection", logger.Error(err))
	}

	a.logger.Info("agent stopped")

	return nil
}

func (a *Agent) acceptLoop() {
	defer a.logger.Info("accept loop stopped")

	handler := NewConnectionHandler(a.tracker, a.serverConn.GetPacketChannel(), a.logger)

	for {
		conn, err := a.listener.Accept()
		if err != nil {
			select {
			case <-a.serverConn.stopChan:
				a.logger.Debug("listener closed")
				return
			default:
			}

			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				a.logger.Debug("listener closed")
				return
			}

			a.logger.Error("accept error", logger.Error(err))
			continue
		}

		go handler.Handle(conn)
	}
}

func (a *Agent) cleanupLoop() {
	defer a.logger.Info("cleanup loop stopped")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		removed := a.tracker.Cleanup(5 * time.Minute)
		if removed > 0 {
			a.logger.Info("cleanup cycle completed",
				logger.Int("removed", removed),
				logger.Int("active", a.tracker.Count()),
			)
		}
	}
}
