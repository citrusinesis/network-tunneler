package handler

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"google.golang.org/protobuf/proto"

	pb "network-tunneler/proto"

	"network-tunneler/pkg/logger"
)

type Agent struct {
	registry RegistryAPI
	router   RouterAPI
	logger   logger.Logger
}

func NewAgent(registry RegistryAPI, router RouterAPI, log logger.Logger) *Agent {
	return &Agent{
		registry: registry,
		router:   router,
		logger:   log.With(logger.String("handler", "agent")),
	}
}

func (h *Agent) Handle(conn net.Conn) {
	defer conn.Close()

	h.logger.Debug("handling agent connection",
		logger.String("remote", conn.RemoteAddr().String()),
	)

	register, err := h.readMessage(conn)
	if err != nil {
		h.logger.Error("failed to read registration", logger.Error(err))
		return
	}

	agentReg, ok := register.(*pb.AgentRegister)
	if !ok {
		h.logger.Error("expected AgentRegister message")
		return
	}

	agentID := agentReg.AgentId
	h.logger.Info("agent registering", logger.String("agent_id", agentID))

	if err := h.registry.RegisterAgent(agentID, conn); err != nil {
		h.logger.Error("failed to register agent",
			logger.String("agent_id", agentID),
			logger.Error(err),
		)

		ack := &pb.RegisterAck{
			Success: false,
			Message: err.Error(),
		}
		h.sendMessage(conn, ack)
		return
	}
	defer h.registry.UnregisterAgent(agentID)

	ack := &pb.RegisterAck{
		Success: true,
		Message: "registered successfully",
	}
	if err := h.sendMessage(conn, ack); err != nil {
		h.logger.Error("failed to send ack", logger.Error(err))
		return
	}

	h.logger.Info("agent registered successfully", logger.String("agent_id", agentID))

	for {
		msg, err := h.readMessage(conn)
		if err != nil {
			if err == io.EOF {
				h.logger.Info("agent disconnected", logger.String("agent_id", agentID))
			} else {
				h.logger.Error("failed to read message",
					logger.String("agent_id", agentID),
					logger.Error(err),
				)
			}
			return
		}

		h.handleMessage(agentID, msg)
	}
}

func (h *Agent) handleMessage(agentID string, msg any) {
	switch m := msg.(type) {
	case *pb.Packet:
		h.logger.Debug("received packet from agent",
			logger.String("agent_id", agentID),
			logger.String("conn_id", m.ConnectionId),
		)

		if err := h.router.RouteFromAgent(m); err != nil {
			h.logger.Error("failed to route packet",
				logger.String("agent_id", agentID),
				logger.String("conn_id", m.ConnectionId),
				logger.Error(err),
			)
		}

	case *pb.Heartbeat:
		h.logger.Debug("received heartbeat from agent",
			logger.String("agent_id", agentID),
		)

	default:
		h.logger.Warn("unknown message type from agent",
			logger.String("agent_id", agentID),
		)
	}
}

func (h *Agent) readMessage(conn net.Conn) (any, error) {
	var msgType uint8
	if err := binary.Read(conn, binary.BigEndian, &msgType); err != nil {
		return nil, err
	}

	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	if msgLen > 1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", msgLen)
	}

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	switch pb.MessageType(msgType) {
	case pb.MessageType_AGENT_REGISTER:
		msg := &pb.AgentRegister{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil

	case pb.MessageType_PACKET:
		msg := &pb.Packet{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil

	case pb.MessageType_HEARTBEAT:
		msg := &pb.Heartbeat{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil

	default:
		return nil, fmt.Errorf("unknown message type: %d", msgType)
	}
}

func (h *Agent) sendMessage(conn net.Conn, msg proto.Message) error {
	var msgType pb.MessageType
	switch msg.(type) {
	case *pb.RegisterAck:
		msgType = pb.MessageType_REGISTER_ACK
	case *pb.Packet:
		msgType = pb.MessageType_PACKET
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := binary.Write(conn, binary.BigEndian, uint8(msgType)); err != nil {
		return err
	}

	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}

	return nil
}
