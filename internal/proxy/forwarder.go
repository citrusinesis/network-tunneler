package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ConnectionState struct {
	ConnectionID string
	TargetAddr   string
	TargetConn   net.Conn
	CreatedAt    time.Time
	LastActivity time.Time
}

type PacketForwarder struct {
	logger       logger.Logger
	responseChan chan<- *pb.Packet
	connections  map[string]*ConnectionState
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

type ForwarderParams struct {
	Logger       logger.Logger
	ResponseChan chan<- *pb.Packet
}

func NewPacketForwarder(p ForwarderParams) *PacketForwarder {
	ctx, cancel := context.WithCancel(context.Background())
	return &PacketForwarder{
		logger:       p.Logger.With(logger.String("component", "forwarder")),
		responseChan: p.ResponseChan,
		connections:  make(map[string]*ConnectionState),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (pf *PacketForwarder) Forward(pkt *pb.Packet) error {
	pf.mu.Lock()
	state, exists := pf.connections[pkt.ConnectionId]
	if !exists {
		targetAddr := net.JoinHostPort(pkt.ConnTuple.DstIp, fmt.Sprintf("%d", pkt.ConnTuple.DstPort))

		conn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
		if err != nil {
			pf.mu.Unlock()
			return fmt.Errorf("failed to dial target %s: %w", targetAddr, err)
		}

		state = &ConnectionState{
			ConnectionID: pkt.ConnectionId,
			TargetAddr:   targetAddr,
			TargetConn:   conn,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}

		pf.connections[pkt.ConnectionId] = state

		pf.logger.Info("new target connection established",
			logger.String("conn_id", pkt.ConnectionId),
			logger.String("target", targetAddr),
		)

		pf.wg.Add(1)
		go pf.readFromTarget(state)
	} else {
		state.LastActivity = time.Now()
	}
	pf.mu.Unlock()

	_, err := state.TargetConn.Write(pkt.Data)
	if err != nil {
		pf.removeConnection(pkt.ConnectionId)
		return fmt.Errorf("failed to write to target: %w", err)
	}

	pf.logger.Debug("packet forwarded to target",
		logger.String("conn_id", pkt.ConnectionId),
		logger.Int("bytes", len(pkt.Data)),
	)

	return nil
}

func (pf *PacketForwarder) readFromTarget(state *ConnectionState) {
	defer pf.wg.Done()
	defer pf.removeConnection(state.ConnectionID)

	buf := make([]byte, 65535)

	for {
		select {
		case <-pf.ctx.Done():
			pf.logger.Debug("context cancelled, stopping read loop",
				logger.String("conn_id", state.ConnectionID),
			)
			return
		default:
		}

		state.TargetConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		n, err := state.TargetConn.Read(buf)
		if err != nil {
			if err == io.EOF {
				pf.logger.Debug("target connection closed",
					logger.String("conn_id", state.ConnectionID),
				)
			} else {
				pf.logger.Error("read error from target",
					logger.String("conn_id", state.ConnectionID),
					logger.Error(err),
				)
			}
			return
		}

		if n == 0 {
			continue
		}

		pf.mu.Lock()
		state.LastActivity = time.Now()
		pf.mu.Unlock()

		responsePkt := &pb.Packet{
			ConnectionId: state.ConnectionID,
			Data:         append([]byte(nil), buf[:n]...),
			Protocol:     pb.Protocol_PROTOCOL_TCP,
			Direction:    pb.Direction_DIRECTION_REVERSE,
			Timestamp:    time.Now().Unix(),
		}

		select {
		case pf.responseChan <- responsePkt:
		case <-pf.ctx.Done():
			pf.logger.Debug("context cancelled while sending packet",
				logger.String("conn_id", state.ConnectionID),
			)
			return
		}

		pf.logger.Debug("response sent to server",
			logger.String("conn_id", state.ConnectionID),
			logger.Int("bytes", n),
		)
	}
}

func (pf *PacketForwarder) removeConnection(connID string) {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if state, exists := pf.connections[connID]; exists {
		state.TargetConn.Close()
		delete(pf.connections, connID)

		pf.logger.Debug("connection removed",
			logger.String("conn_id", connID),
		)
	}
}

func (pf *PacketForwarder) Cleanup(maxIdleTime time.Duration) int {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	now := time.Now()
	removed := 0

	for connID, state := range pf.connections {
		if now.Sub(state.LastActivity) > maxIdleTime {
			state.TargetConn.Close()
			delete(pf.connections, connID)
			removed++

			pf.logger.Debug("idle connection cleaned up",
				logger.String("conn_id", connID),
				logger.Duration("idle_time", now.Sub(state.LastActivity)),
			)
		}
	}

	if removed > 0 {
		pf.logger.Info("cleanup completed",
			logger.Int("removed_connections", removed),
		)
	}

	return removed
}

func (pf *PacketForwarder) Count() int {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	return len(pf.connections)
}

func (pf *PacketForwarder) Stop() {
	pf.cancel()
	pf.wg.Wait()

	pf.logger.Info("packet forwarder stopped")
}
