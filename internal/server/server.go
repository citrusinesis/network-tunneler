package server

import (
	"context"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
)

type Server struct {
	cfg        *Config
	logger     logger.Logger
	registry   *Registry
	grpcServer *GRPCServer
}

type Params struct {
	fx.In

	Config     *Config
	Logger     logger.Logger
	Registry   *Registry
	GRPCServer *GRPCServer
}

func New(lc fx.Lifecycle, p Params) *Server {
	s := &Server{
		cfg:        p.Config,
		logger:     p.Logger.With(logger.String("component", "server")),
		registry:   p.Registry,
		grpcServer: p.GRPCServer,
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

func (s *Server) start(ctx context.Context) error {
	s.logger.Info("starting server",
		logger.String("client_addr", s.cfg.ClientListenAddr),
		logger.String("proxy_addr", s.cfg.ProxyListenAddr),
	)

	if err := s.grpcServer.Start(ctx); err != nil {
		return err
	}

	s.logger.Info("server started successfully")
	return nil
}

func (s *Server) stop(ctx context.Context) error {
	s.logger.Info("stopping server")

	if err := s.grpcServer.Stop(ctx); err != nil {
		s.logger.Warn("grpc server stop error", logger.Error(err))
	}

	if err := s.registry.Cleanup(ctx); err != nil {
		s.logger.Warn("registry cleanup error", logger.Error(err))
	}

	s.logger.Info("server stopped")
	return nil
}
