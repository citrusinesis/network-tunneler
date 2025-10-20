package server

import (
	"io"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ClientService struct {
	pb.UnimplementedTunnelClientServer
	registry *Registry
	logger   logger.Logger
}

func NewClientService(registry *Registry, log logger.Logger) *ClientService {
	return &ClientService{
		registry: registry,
		logger:   log.With(logger.String("service", "client")),
	}
}

func (s *ClientService) Connect(stream pb.TunnelClient_ConnectServer) error {
	var clientID string
	var registered bool

	s.logger.Debug("new client connection stream")

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("client stream context cancelled",
				logger.String("client_id", clientID),
			)
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("client disconnected", logger.String("client_id", clientID))
			return nil
		}
		if err != nil {
			s.logger.Error("stream recv error",
				logger.String("client_id", clientID),
				logger.Error(err),
			)
			return err
		}

		switch m := msg.Message.(type) {
		case *pb.ClientMessage_Register:
			clientID = m.Register.ClientId
			s.logger.Info("client registering", logger.String("client_id", clientID))

			ack := &pb.RegisterAck{
				Success: true,
				Message: "registered successfully",
			}

			if err := s.registry.RegisterClientStream(clientID, stream); err != nil {
				s.logger.Error("failed to register client",
					logger.String("client_id", clientID),
					logger.Error(err),
				)
				ack.Success = false
				ack.Message = err.Error()
			} else {
				registered = true
				defer s.registry.UnregisterClient(clientID)
			}

			if err := stream.Send(&pb.ClientMessage{
				Message: &pb.ClientMessage_Ack{Ack: ack},
			}); err != nil {
				s.logger.Error("failed to send ack", logger.Error(err))
				return err
			}

		case *pb.ClientMessage_Packet:
			if !registered {
				s.logger.Warn("packet from unregistered client")
				continue
			}

			s.logger.Debug("received packet from client",
				logger.String("client_id", clientID),
				logger.String("conn_id", m.Packet.ConnectionId),
			)

			if err := s.registry.RouteFromClient(clientID, m.Packet); err != nil {
				s.logger.Error("failed to route packet",
					logger.String("client_id", clientID),
					logger.String("conn_id", m.Packet.ConnectionId),
					logger.Error(err),
				)
			}

		case *pb.ClientMessage_Heartbeat:
			if !registered {
				s.logger.Warn("heartbeat from unregistered client")
				continue
			}

			s.logger.Debug("received heartbeat from client",
				logger.String("client_id", clientID),
			)

		default:
			s.logger.Warn("unknown message type from client",
				logger.String("client_id", clientID),
			)
		}
	}
}
