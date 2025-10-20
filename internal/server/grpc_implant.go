package server

import (
	"io"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ImplantService struct {
	pb.UnimplementedTunnelImplantServer
	registry *Registry
	logger   logger.Logger
}

func NewImplantService(registry *Registry, log logger.Logger) *ImplantService {
	return &ImplantService{
		registry: registry,
		logger:   log.With(logger.String("service", "implant")),
	}
}

func (s *ImplantService) Connect(stream pb.TunnelImplant_ConnectServer) error {
	var implantID string
	var managedCIDR string
	var registered bool

	s.logger.Debug("new implant connection stream")

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("implant stream context cancelled",
				logger.String("implant_id", implantID),
			)
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("implant disconnected",
				logger.String("implant_id", implantID),
			)
			return nil
		}
		if err != nil {
			s.logger.Error("stream recv error",
				logger.String("implant_id", implantID),
				logger.Error(err),
			)
			return err
		}

		switch m := msg.Message.(type) {
		case *pb.ImplantMessage_Register:
			implantID = m.Register.ImplantId
			managedCIDR = m.Register.ManagedCidr

			s.logger.Info("implant registering",
				logger.String("implant_id", implantID),
				logger.String("managed_cidr", managedCIDR),
			)

			ack := &pb.RegisterAck{
				Success: true,
				Message: "registered successfully",
			}

			if err := s.registry.RegisterImplantStream(implantID, stream, managedCIDR); err != nil {
				s.logger.Error("failed to register implant",
					logger.String("implant_id", implantID),
					logger.Error(err),
				)
				ack.Success = false
				ack.Message = err.Error()
			} else {
				registered = true
				defer s.registry.UnregisterImplant(implantID)
			}

			if err := stream.Send(&pb.ImplantMessage{
				Message: &pb.ImplantMessage_Ack{Ack: ack},
			}); err != nil {
				s.logger.Error("failed to send ack", logger.Error(err))
				return err
			}

		case *pb.ImplantMessage_Packet:
			if !registered {
				s.logger.Warn("packet from unregistered implant")
				continue
			}

			s.logger.Debug("received packet from implant",
				logger.String("implant_id", implantID),
				logger.String("conn_id", m.Packet.ConnectionId),
			)

			if err := s.registry.RouteFromImplant(implantID, m.Packet); err != nil {
				s.logger.Error("failed to route packet",
					logger.String("implant_id", implantID),
					logger.String("conn_id", m.Packet.ConnectionId),
					logger.Error(err),
				)
			}

		case *pb.ImplantMessage_Heartbeat:
			if !registered {
				s.logger.Warn("heartbeat from unregistered implant")
				continue
			}

			s.logger.Debug("received heartbeat from implant",
				logger.String("implant_id", implantID),
			)

		default:
			s.logger.Warn("unknown message type from implant",
				logger.String("implant_id", implantID),
			)
		}
	}
}
