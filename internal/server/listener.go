package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"network-tunneler/pkg/logger"
)

type ConnectionHandler func(net.Conn)

type ListenerManager struct {
	agentListener   net.Listener
	implantListener net.Listener
	wg              sync.WaitGroup
	logger          logger.Logger
	ctx             context.Context
	cancel          context.CancelFunc
}

func NewListenerManager(log logger.Logger) *ListenerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ListenerManager{
		logger: log.With(logger.String("component", "listener")),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (l *ListenerManager) Start(
	agentAddr, implantAddr string,
	tlsConfig *tls.Config,
	onAgent, onImplant ConnectionHandler,
) error {
	agentLn, err := tls.Listen("tcp", agentAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to listen for agents: %w", err)
	}
	l.agentListener = agentLn

	implantLn, err := tls.Listen("tcp", implantAddr, tlsConfig)
	if err != nil {
		agentLn.Close()
		return fmt.Errorf("failed to listen for implants: %w", err)
	}
	l.implantListener = implantLn

	l.logger.Info("listeners started",
		logger.String("agent_addr", agentAddr),
		logger.String("implant_addr", implantAddr),
	)

	l.wg.Add(2)
	go l.acceptLoop(l.agentListener, "agent", onAgent)
	go l.acceptLoop(l.implantListener, "implant", onImplant)

	return nil
}

func (l *ListenerManager) Stop(ctx context.Context) error {
	l.logger.Info("stopping listeners")

	l.cancel()

	if l.agentListener != nil {
		l.agentListener.Close()
	}
	if l.implantListener != nil {
		l.implantListener.Close()
	}

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		l.logger.Info("listeners stopped gracefully")
		return nil
	case <-ctx.Done():
		l.logger.Warn("listener stop timeout")
		return ctx.Err()
	}
}

func (l *ListenerManager) acceptLoop(listener net.Listener, name string, handler ConnectionHandler) {
	defer l.wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-l.ctx.Done():
				l.logger.Debug("listener closed", logger.String("listener", name))
				return
			default:
			}

			l.logger.Warn("accept error",
				logger.String("listener", name),
				logger.Error(err),
			)
			continue
		}

		l.logger.Info("connection accepted",
			logger.String("listener", name),
			logger.String("remote", conn.RemoteAddr().String()),
		)

		go handler(conn)
	}
}
