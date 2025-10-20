package server

import (
	"io"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type AgentService struct {
	pb.UnimplementedTunnelAgentServer
	registry *Registry
	logger   logger.Logger
}

func NewAgentService(registry *Registry, log logger.Logger) *AgentService {
	return &AgentService{
		registry: registry,
		logger:   log.With(logger.String("service", "agent")),
	}
}

func (s *AgentService) Connect(stream pb.TunnelAgent_ConnectServer) error {
	var agentID string
	var registered bool

	s.logger.Debug("new agent connection stream")

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("agent stream context cancelled",
				logger.String("agent_id", agentID),
			)
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("agent disconnected", logger.String("agent_id", agentID))
			return nil
		}
		if err != nil {
			s.logger.Error("stream recv error",
				logger.String("agent_id", agentID),
				logger.Error(err),
			)
			return err
		}

		switch m := msg.Message.(type) {
		case *pb.AgentMessage_Register:
			agentID = m.Register.AgentId
			s.logger.Info("agent registering", logger.String("agent_id", agentID))

			ack := &pb.RegisterAck{
				Success: true,
				Message: "registered successfully",
			}

			if err := s.registry.RegisterAgentStream(agentID, stream); err != nil {
				s.logger.Error("failed to register agent",
					logger.String("agent_id", agentID),
					logger.Error(err),
				)
				ack.Success = false
				ack.Message = err.Error()
			} else {
				registered = true
				defer s.registry.UnregisterAgent(agentID)
			}

			if err := stream.Send(&pb.AgentMessage{
				Message: &pb.AgentMessage_Ack{Ack: ack},
			}); err != nil {
				s.logger.Error("failed to send ack", logger.Error(err))
				return err
			}

		case *pb.AgentMessage_Packet:
			if !registered {
				s.logger.Warn("packet from unregistered agent")
				continue
			}

			s.logger.Debug("received packet from agent",
				logger.String("agent_id", agentID),
				logger.String("conn_id", m.Packet.ConnectionId),
			)

			if err := s.registry.RouteFromAgent(agentID, m.Packet); err != nil {
				s.logger.Error("failed to route packet",
					logger.String("agent_id", agentID),
					logger.String("conn_id", m.Packet.ConnectionId),
					logger.Error(err),
				)
			}

		case *pb.AgentMessage_Heartbeat:
			if !registered {
				s.logger.Warn("heartbeat from unregistered agent")
				continue
			}

			s.logger.Debug("received heartbeat from agent",
				logger.String("agent_id", agentID),
			)

		default:
			s.logger.Warn("unknown message type from agent",
				logger.String("agent_id", agentID),
			)
		}
	}
}
