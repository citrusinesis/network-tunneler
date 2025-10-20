package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

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
	clientService   *ClientService
	proxyService *ProxyService

	clientServer   *grpc.Server
	proxyServer *grpc.Server

	wg sync.WaitGroup
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
		clientService:   NewClientService(registry, log),
		proxyService: NewProxyService(registry, log),
	}
}

func (s *GRPCServer) Start(ctx context.Context) error {
	creds := credentials.NewTLS(s.tlsConfig)

	s.clientServer = grpc.NewServer(grpc.Creds(creds))
	pb.RegisterTunnelClientServer(s.clientServer, s.clientService)

	s.proxyServer = grpc.NewServer(grpc.Creds(creds))
	pb.RegisterTunnelProxyServer(s.proxyServer, s.proxyService)

	clientLis, err := net.Listen("tcp", s.cfg.ClientListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen for clients: %w", err)
	}

	proxyLis, err := net.Listen("tcp", s.cfg.ProxyListenAddr)
	if err != nil {
		clientLis.Close()
		return fmt.Errorf("failed to listen for proxys: %w", err)
	}

	s.logger.Info("gRPC servers starting",
		logger.String("client_addr", s.cfg.ClientListenAddr),
		logger.String("proxy_addr", s.cfg.ProxyListenAddr),
	)

	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		if err := s.clientServer.Serve(clientLis); err != nil {
			s.logger.Error("client gRPC server error", logger.Error(err))
		}
		s.logger.Debug("client gRPC server goroutine stopped")
	}()

	go func() {
		defer s.wg.Done()
		if err := s.proxyServer.Serve(proxyLis); err != nil {
			s.logger.Error("proxy gRPC server error", logger.Error(err))
		}
		s.logger.Debug("proxy gRPC server goroutine stopped")
	}()

	s.logger.Info("gRPC servers started successfully")
	return nil
}

func (s *GRPCServer) Stop(ctx context.Context) error {
	s.logger.Info("stopping gRPC servers")

	// Use a goroutine to perform graceful stop with context timeout protection
	done := make(chan struct{})
	go func() {
		if s.clientServer != nil {
			s.clientServer.GracefulStop()
		}
		if s.proxyServer != nil {
			s.proxyServer.GracefulStop()
		}
		close(done)
	}()

	// Wait for graceful stop or context timeout
	select {
	case <-done:
		s.logger.Info("gRPC servers stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("context timeout during graceful stop, forcing shutdown")
		// Force stop if graceful stop times out
		if s.clientServer != nil {
			s.clientServer.Stop()
		}
		if s.proxyServer != nil {
			s.proxyServer.Stop()
		}
	}

	// Wait for server goroutines to complete
	s.wg.Wait()

	s.logger.Info("all gRPC server goroutines stopped")
	return nil
}
