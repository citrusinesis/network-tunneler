package server

import (
	"io"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ProxyService struct {
	pb.UnimplementedTunnelProxyServer
	registry *Registry
	logger   logger.Logger
}

func NewProxyService(registry *Registry, log logger.Logger) *ProxyService {
	return &ProxyService{
		registry: registry,
		logger:   log.With(logger.String("service", "proxy")),
	}
}

func (s *ProxyService) Connect(stream pb.TunnelProxy_ConnectServer) error {
	var proxyID string
	var managedCIDR string
	var registered bool

	s.logger.Debug("new proxy connection stream")

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("proxy stream context cancelled",
				logger.String("proxy_id", proxyID),
			)
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("proxy disconnected",
				logger.String("proxy_id", proxyID),
			)
			return nil
		}
		if err != nil {
			s.logger.Error("stream recv error",
				logger.String("proxy_id", proxyID),
				logger.Error(err),
			)
			return err
		}

		switch m := msg.Message.(type) {
		case *pb.ProxyMessage_Register:
			proxyID = m.Register.ProxyId
			managedCIDR = m.Register.ManagedCidr

			s.logger.Info("proxy registering",
				logger.String("proxy_id", proxyID),
				logger.String("managed_cidr", managedCIDR),
			)

			ack := &pb.RegisterAck{
				Success: true,
				Message: "registered successfully",
			}

			if err := s.registry.RegisterProxyStream(proxyID, stream, managedCIDR); err != nil {
				s.logger.Error("failed to register proxy",
					logger.String("proxy_id", proxyID),
					logger.Error(err),
				)
				ack.Success = false
				ack.Message = err.Error()
			} else {
				registered = true
				defer s.registry.UnregisterProxy(proxyID)
			}

			if err := stream.Send(&pb.ProxyMessage{
				Message: &pb.ProxyMessage_Ack{Ack: ack},
			}); err != nil {
				s.logger.Error("failed to send ack", logger.Error(err))
				return err
			}

		case *pb.ProxyMessage_Packet:
			if !registered {
				s.logger.Warn("packet from unregistered proxy")
				continue
			}

			s.logger.Debug("received packet from proxy",
				logger.String("proxy_id", proxyID),
				logger.String("conn_id", m.Packet.ConnectionId),
			)

			if err := s.registry.RouteFromProxy(proxyID, m.Packet); err != nil {
				s.logger.Error("failed to route packet",
					logger.String("proxy_id", proxyID),
					logger.String("conn_id", m.Packet.ConnectionId),
					logger.Error(err),
				)
			}

		case *pb.ProxyMessage_Heartbeat:
			if !registered {
				s.logger.Warn("heartbeat from unregistered proxy")
				continue
			}

			s.logger.Debug("received heartbeat from proxy",
				logger.String("proxy_id", proxyID),
			)

		default:
			s.logger.Warn("unknown message type from proxy",
				logger.String("proxy_id", proxyID),
			)
		}
	}
}
