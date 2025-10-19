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

type Implant struct {
	registry RegistryAPI
	router   RouterAPI
	logger   logger.Logger
}

func NewImplant(registry RegistryAPI, router RouterAPI, log logger.Logger) *Implant {
	return &Implant{
		registry: registry,
		router:   router,
		logger:   log.With(logger.String("handler", "implant")),
	}
}

func (h *Implant) Handle(conn net.Conn) {
	defer conn.Close()

	h.logger.Debug("handling implant connection",
		logger.String("remote", conn.RemoteAddr().String()),
	)

	register, err := h.readMessage(conn)
	if err != nil {
		h.logger.Error("failed to read registration", logger.Error(err))
		return
	}

	implantReg, ok := register.(*pb.ImplantRegister)
	if !ok {
		h.logger.Error("expected ImplantRegister message")
		return
	}

	implantID := implantReg.ImplantId
	managedCIDR := implantReg.ManagedCidr

	h.logger.Info("implant registering",
		logger.String("implant_id", implantID),
		logger.String("managed_cidr", managedCIDR),
	)

	if err := h.registry.RegisterImplant(implantID, conn, managedCIDR); err != nil {
		h.logger.Error("failed to register implant",
			logger.String("implant_id", implantID),
			logger.Error(err),
		)

		ack := &pb.RegisterAck{
			Success: false,
			Message: err.Error(),
		}
		h.sendMessage(conn, ack)
		return
	}
	defer h.registry.UnregisterImplant(implantID)

	ack := &pb.RegisterAck{
		Success: true,
		Message: "registered successfully",
	}
	if err := h.sendMessage(conn, ack); err != nil {
		h.logger.Error("failed to send ack", logger.Error(err))
		return
	}

	h.logger.Info("implant registered successfully",
		logger.String("implant_id", implantID),
		logger.String("managed_cidr", managedCIDR),
	)

	for {
		msg, err := h.readMessage(conn)
		if err != nil {
			if err == io.EOF {
				h.logger.Info("implant disconnected", logger.String("implant_id", implantID))
			} else {
				h.logger.Error("failed to read message",
					logger.String("implant_id", implantID),
					logger.Error(err),
				)
			}
			return
		}

		h.handleMessage(implantID, msg)
	}
}

func (h *Implant) handleMessage(implantID string, msg any) {
	switch m := msg.(type) {
	case *pb.Packet:
		h.logger.Debug("received packet from implant",
			logger.String("implant_id", implantID),
			logger.String("conn_id", m.ConnectionId),
		)

		if err := h.router.RouteFromImplant(m); err != nil {
			h.logger.Error("failed to route packet",
				logger.String("implant_id", implantID),
				logger.String("conn_id", m.ConnectionId),
				logger.Error(err),
			)
		}

	case *pb.Heartbeat:
		h.logger.Debug("received heartbeat from implant",
			logger.String("implant_id", implantID),
		)

	default:
		h.logger.Warn("unknown message type from implant",
			logger.String("implant_id", implantID),
		)
	}
}

func (h *Implant) readMessage(conn net.Conn) (any, error) {
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
	case pb.MessageType_IMPLANT_REGISTER:
		msg := &pb.ImplantRegister{}
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

func (h *Implant) sendMessage(conn net.Conn, msg proto.Message) error {
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
