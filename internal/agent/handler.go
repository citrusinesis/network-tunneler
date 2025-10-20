package agent

import (
	"io"
	"net"
	"time"

	"network-tunneler/pkg/logger"
	pkgnet "network-tunneler/pkg/network"
	"network-tunneler/proto"
)

type OriginalDestFunc func(net.Conn) (string, error)

type ConnectionHandler struct {
	tracker         *ConnectionTracker
	serverWriter    chan<- *proto.Packet
	logger          logger.Logger
	getOriginalDest OriginalDestFunc
}

func NewConnectionHandler(tracker *ConnectionTracker, serverWriter chan<- *proto.Packet, log logger.Logger) *ConnectionHandler {
	return &ConnectionHandler{
		tracker:         tracker,
		serverWriter:    serverWriter,
		logger:          log.With(logger.String("component", "handler")),
		getOriginalDest: pkgnet.GetOriginalDestAuto,
	}
}

func (h *ConnectionHandler) Handle(conn net.Conn) {
	defer conn.Close()

	originalDest, err := h.getOriginalDest(conn)
	if err != nil {
		h.logger.Error("failed to get original destination",
			logger.Error(err),
			logger.String("remote_addr", conn.RemoteAddr().String()),
		)
		return
	}

	srcHost, srcPort, err := parseAddr(conn.RemoteAddr().String())
	if err != nil {
		h.logger.Error("failed to parse source address",
			logger.Error(err),
		)
		return
	}

	dstHost, dstPort, err := parseAddr(originalDest)
	if err != nil {
		h.logger.Error("failed to parse destination address",
			logger.Error(err),
		)
		return
	}

	srcIP := net.ParseIP(srcHost)
	dstIP := net.ParseIP(dstHost)

	connID := pkgnet.GenerateConnectionID(srcIP, srcPort, dstIP, dstPort)

	h.tracker.Track(connID, originalDest, conn)
	defer h.tracker.Remove(connID)

	h.logger.Info("new connection",
		logger.String("connection_id", connID),
		logger.String("source", conn.RemoteAddr().String()),
		logger.String("original_dest", originalDest),
	)

	buf := make([]byte, 65535)
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				h.logger.Debug("connection closed by client",
					logger.String("connection_id", connID),
				)
			} else {
				h.logger.Error("read error",
					logger.Error(err),
					logger.String("connection_id", connID),
				)
			}
			return
		}

		if n == 0 {
			continue
		}

		h.tracker.UpdateActivity(connID)

		packet := &proto.Packet{
			ConnectionId: connID,
			Data:         buf[:n],
			ConnTuple: &proto.ConnectionTuple{
				SrcIp:   srcIP.String(),
				SrcPort: uint32(srcPort),
				DstIp:   dstIP.String(),
				DstPort: uint32(dstPort),
			},
			Protocol:  proto.Protocol_PROTOCOL_TCP,
			Direction: proto.Direction_DIRECTION_FORWARD,
			Timestamp: time.Now().Unix(),
		}

		select {
		case h.serverWriter <- packet:
			h.logger.Debug("packet sent to server",
				logger.String("connection_id", connID),
				logger.Int("bytes", n),
			)
		default:
			h.logger.Warn("server writer channel full, dropping packet",
				logger.String("connection_id", connID),
			)
		}
	}
}

func parseAddr(addr string) (string, uint16, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}

	portNum, err := net.LookupPort("tcp", port)
	if err != nil {
		return "", 0, err
	}

	return host, uint16(portNum), nil
}
