package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"go.uber.org/fx"

	"network-tunneler/pkg/logger"
	"network-tunneler/proto"
)

type Connection struct {
	AgentConn   net.Conn
	ImplantConn net.Conn
	ConnID      string
}

type Router struct {
	logger logger.Logger

	mu          sync.RWMutex
	connections map[string]*Connection
}

type RouterParams struct {
	fx.In

	Logger logger.Logger
}

func NewRouter(lc fx.Lifecycle, p RouterParams) *Router {
	r := &Router{
		logger:      p.Logger.With(logger.String("component", "router")),
		connections: make(map[string]*Connection),
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return r.cleanup(ctx)
		},
	})

	return r
}

func (r *Router) cleanup(_ context.Context) error {
	r.logger.Info("cleaning up connections")

	r.mu.Lock()
	defer r.mu.Unlock()

	for connID, conn := range r.connections {
		r.logger.Debug("closing connection", logger.String("conn_id", connID))
		if conn.AgentConn != nil {
			conn.AgentConn.Close()
		}
		if conn.ImplantConn != nil {
			conn.ImplantConn.Close()
		}
	}

	r.connections = make(map[string]*Connection)
	return nil
}

func (r *Router) RegisterAgent(connID string, agentConn net.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conn, exists := r.connections[connID]; exists {
		conn.AgentConn = agentConn
		r.logger.Debug("agent registered to existing connection", logger.String("conn_id", connID))
		return nil
	}

	r.connections[connID] = &Connection{
		AgentConn: agentConn,
		ConnID:    connID,
	}

	r.logger.Debug("new connection created for agent", logger.String("conn_id", connID))
	return nil
}

func (r *Router) RegisterImplant(connID string, implantConn net.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conn, exists := r.connections[connID]; exists {
		conn.ImplantConn = implantConn
		r.logger.Debug("implant registered to existing connection", logger.String("conn_id", connID))
		return nil
	}

	r.connections[connID] = &Connection{
		ImplantConn: implantConn,
		ConnID:      connID,
	}

	r.logger.Debug("new connection created for implant", logger.String("conn_id", connID))
	return nil
}

func (r *Router) RouteFromAgent(pkt *proto.Packet) error {
	r.mu.RLock()
	conn, exists := r.connections[pkt.ConnectionId]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", pkt.ConnectionId)
	}

	if conn.ImplantConn == nil {
		return fmt.Errorf("no implant for connection: %s", pkt.ConnectionId)
	}

	r.logger.Debug("routing packet from agent to implant",
		logger.String("conn_id", pkt.ConnectionId),
		logger.Int("size", len(pkt.Data)),
	)

	return nil
}

func (r *Router) RouteFromImplant(pkt *proto.Packet) error {
	r.mu.RLock()
	conn, exists := r.connections[pkt.ConnectionId]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", pkt.ConnectionId)
	}

	if conn.AgentConn == nil {
		return fmt.Errorf("no agent for connection: %s", pkt.ConnectionId)
	}

	r.logger.Debug("routing packet from implant to agent",
		logger.String("conn_id", pkt.ConnectionId),
		logger.Int("size", len(pkt.Data)),
	)

	return nil
}

func (r *Router) RemoveConnection(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.connections, connID)
	r.logger.Debug("connection removed", logger.String("conn_id", connID))
}

func (r *Router) GetConnectionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connections)
}
