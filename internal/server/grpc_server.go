package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type GRPCServer struct {
	cfg            *Config
	logger         logger.Logger
	tlsConfig      *tls.Config
	registry       *Registry
	agentService   *AgentService
	implantService *ImplantService

	agentServer   *grpc.Server
	implantServer *grpc.Server
}

func NewGRPCServer(
	cfg *Config,
	log logger.Logger,
	tlsConfig *tls.Config,
	registry *Registry,
) *GRPCServer {
	return &GRPCServer{
		cfg:            cfg,
		logger:         log.With(logger.String("component", "grpc-server")),
		tlsConfig:      tlsConfig,
		registry:       registry,
		agentService:   NewAgentService(registry, log),
		implantService: NewImplantService(registry, log),
	}
}

func (s *GRPCServer) Start(ctx context.Context) error {
	creds := credentials.NewTLS(s.tlsConfig)

	s.agentServer = grpc.NewServer(grpc.Creds(creds))
	pb.RegisterTunnelAgentServer(s.agentServer, s.agentService)

	s.implantServer = grpc.NewServer(grpc.Creds(creds))
	pb.RegisterTunnelImplantServer(s.implantServer, s.implantService)

	agentLis, err := net.Listen("tcp", s.cfg.AgentListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen for agents: %w", err)
	}

	implantLis, err := net.Listen("tcp", s.cfg.ImplantListenAddr)
	if err != nil {
		agentLis.Close()
		return fmt.Errorf("failed to listen for implants: %w", err)
	}

	s.logger.Info("gRPC servers starting",
		logger.String("agent_addr", s.cfg.AgentListenAddr),
		logger.String("implant_addr", s.cfg.ImplantListenAddr),
	)

	go func() {
		if err := s.agentServer.Serve(agentLis); err != nil {
			s.logger.Error("agent gRPC server error", logger.Error(err))
		}
	}()

	go func() {
		if err := s.implantServer.Serve(implantLis); err != nil {
			s.logger.Error("implant gRPC server error", logger.Error(err))
		}
	}()

	s.logger.Info("gRPC servers started successfully")
	return nil
}

func (s *GRPCServer) Stop(ctx context.Context) error {
	s.logger.Info("stopping gRPC servers")

	if s.agentServer != nil {
		s.agentServer.GracefulStop()
	}
	if s.implantServer != nil {
		s.implantServer.GracefulStop()
	}

	s.logger.Info("gRPC servers stopped")
	return nil
}
