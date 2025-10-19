package server

import (
	"context"

	"go.uber.org/fx"

	"network-tunneler/internal/server/handler"
	"network-tunneler/pkg/logger"
)

type Server struct {
	cfg      *Config
	logger   logger.Logger
	tls      *TLSManager
	listener *ListenerManager
	registry *Registry
	router   *Router

	agentHandler   *handler.Agent
	implantHandler *handler.Implant
}

type Params struct {
	fx.In

	Config         *Config
	Logger         logger.Logger
	TLS            *TLSManager
	Listener       *ListenerManager
	Registry       *Registry
	Router         *Router
	AgentHandler   *handler.Agent
	ImplantHandler *handler.Implant
}

func New(lc fx.Lifecycle, p Params) *Server {
	s := &Server{
		cfg:            p.Config,
		logger:         p.Logger.With(logger.String("component", "server")),
		tls:            p.TLS,
		listener:       p.Listener,
		registry:       p.Registry,
		router:         p.Router,
		agentHandler:   p.AgentHandler,
		implantHandler: p.ImplantHandler,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return s.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return s.stop(ctx)
		},
	})

	return s
}

func (s *Server) start(_ context.Context) error {
	s.logger.Info("starting server",
		logger.String("agent_addr", s.cfg.AgentListenAddr),
		logger.String("implant_addr", s.cfg.ImplantListenAddr),
	)

	tlsConfig, err := s.tls.LoadConfig()
	if err != nil {
		return err
	}

	if err = s.listener.Start(
		s.cfg.AgentListenAddr,
		s.cfg.ImplantListenAddr,
		tlsConfig,
		s.agentHandler.Handle,
		s.implantHandler.Handle,
	); err != nil {
		return err
	}

	s.logger.Info("server started successfully")
	return nil
}

func (s *Server) stop(ctx context.Context) error {
	s.logger.Info("stopping server")

	if err := s.listener.Stop(ctx); err != nil {
		s.logger.Warn("listener stop error", logger.Error(err))
	}

	if err := s.registry.Cleanup(ctx); err != nil {
		s.logger.Warn("registry cleanup error", logger.Error(err))
	}

	s.logger.Info("server stopped")
	return nil
}
