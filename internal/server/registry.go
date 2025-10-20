package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type AgentConn struct {
	ID          string
	Stream      pb.TunnelAgent_ConnectServer
	RemoteAddr  string
	ConnectedAt time.Time
}

type ImplantConn struct {
	ID          string
	Stream      pb.TunnelImplant_ConnectServer
	RemoteAddr  string
	ManagedCIDR string
	ConnectedAt time.Time
}

type Registry struct {
	agents      map[string]*AgentConn
	implants    map[string]*ImplantConn
	connections map[string]*ConnectionRoute // connectionID -> route
	mu          sync.RWMutex
	logger      logger.Logger
}

type ConnectionRoute struct {
	ConnectionID     string
	AgentID          string
	ImplantID        string
	CreatedAt        time.Time
	LastActivity     time.Time
	PacketsToAgent   uint64
	PacketsToImplant uint64
	BytesToAgent     uint64
	BytesToImplant   uint64
}

func NewRegistry(log logger.Logger) *Registry {
	r := &Registry{
		agents:      make(map[string]*AgentConn),
		implants:    make(map[string]*ImplantConn),
		connections: make(map[string]*ConnectionRoute),
		logger:      log.With(logger.String("component", "registry")),
	}

	go r.cleanupLoop()

	return r
}

func (r *Registry) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanupStaleConnections()
	}
}

func (r *Registry) cleanupStaleConnections() {
	const idleTimeout = 5 * time.Minute

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var stale []string

	for connID, route := range r.connections {
		if now.Sub(route.LastActivity) > idleTimeout {
			stale = append(stale, connID)
		}
	}

	for _, connID := range stale {
		delete(r.connections, connID)
		r.logger.Info("cleaned up stale connection",
			logger.String("conn_id", connID),
		)
	}

	if len(stale) > 0 {
		r.logger.Info("connection cleanup completed",
			logger.Int("removed", len(stale)),
			logger.Int("active", len(r.connections)),
		)
	}
}

func (r *Registry) RegisterAgentStream(id string, stream pb.TunnelAgent_ConnectServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		return fmt.Errorf("agent %s already registered", id)
	}

	agent := &AgentConn{
		ID:          id,
		Stream:      stream,
		RemoteAddr:  "grpc-stream",
		ConnectedAt: time.Now(),
	}

	r.agents[id] = agent
	r.logger.Info("agent registered via gRPC",
		logger.String("agent_id", id),
	)

	return nil
}

func (r *Registry) UnregisterAgent(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, exists := r.agents[id]; exists {
		delete(r.agents, id)
		r.logger.Info("agent unregistered",
			logger.String("agent_id", id),
			logger.String("remote", agent.RemoteAddr),
		)
	}
}

func (r *Registry) GetAgent(id string) (*AgentConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[id]
	return agent, exists
}

func (r *Registry) RegisterImplantStream(id string, stream pb.TunnelImplant_ConnectServer, managedCIDR string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.implants[id]; exists {
		return fmt.Errorf("implant %s already registered", id)
	}

	implant := &ImplantConn{
		ID:          id,
		Stream:      stream,
		RemoteAddr:  "grpc-stream",
		ManagedCIDR: managedCIDR,
		ConnectedAt: time.Now(),
	}

	r.implants[id] = implant
	r.logger.Info("implant registered via gRPC",
		logger.String("implant_id", id),
		logger.String("managed_cidr", managedCIDR),
	)

	return nil
}

func (r *Registry) UnregisterImplant(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if implant, exists := r.implants[id]; exists {
		delete(r.implants, id)
		r.logger.Info("implant unregistered",
			logger.String("implant_id", id),
			logger.String("remote", implant.RemoteAddr),
		)
	}
}

func (r *Registry) GetImplant(id string) (*ImplantConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	implant, exists := r.implants[id]
	return implant, exists
}

func (r *Registry) FindImplantByCIDR(targetIP string) (*ImplantConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ip := net.ParseIP(targetIP)
	if ip == nil {
		r.logger.Warn("invalid target IP", logger.String("ip", targetIP))
		return nil, false
	}

	for _, implant := range r.implants {
		_, cidr, err := net.ParseCIDR(implant.ManagedCIDR)
		if err != nil {
			r.logger.Warn("invalid implant CIDR",
				logger.String("implant_id", implant.ID),
				logger.String("cidr", implant.ManagedCIDR),
				logger.Error(err),
			)
			continue
		}

		if cidr.Contains(ip) {
			r.logger.Debug("found implant for target IP",
				logger.String("target_ip", targetIP),
				logger.String("implant_id", implant.ID),
				logger.String("managed_cidr", implant.ManagedCIDR),
			)
			return implant, true
		}
	}

	r.logger.Warn("no implant found for target IP", logger.String("ip", targetIP))
	return nil, false
}

func (r *Registry) ListAgents() []*AgentConn {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*AgentConn, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (r *Registry) ListImplants() []*ImplantConn {
	r.mu.RLock()
	defer r.mu.RUnlock()

	implants := make([]*ImplantConn, 0, len(r.implants))
	for _, implant := range r.implants {
		implants = append(implants, implant)
	}
	return implants
}

func (r *Registry) Cleanup(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("cleaning up registry",
		logger.Int("agents", len(r.agents)),
		logger.Int("implants", len(r.implants)),
		logger.Int("connections", len(r.connections)),
	)

	r.agents = make(map[string]*AgentConn)
	r.implants = make(map[string]*ImplantConn)
	r.connections = make(map[string]*ConnectionRoute)

	return nil
}

func (r *Registry) RouteFromAgent(agentID string, pkt *pb.Packet) error {
	r.mu.Lock()

	route, exists := r.connections[pkt.ConnectionId]
	if !exists {
		destIP := ""
		if pkt.ConnTuple != nil {
			destIP = pkt.ConnTuple.DstIp
		}
		implant, found := r.findImplantByCIDR(destIP)
		if !found {
			r.mu.Unlock()
			return fmt.Errorf("no implant found for destination: %s", destIP)
		}

		now := time.Now()
		route = &ConnectionRoute{
			ConnectionID: pkt.ConnectionId,
			AgentID:      agentID,
			ImplantID:    implant.ID,
			CreatedAt:    now,
			LastActivity: now,
		}
		r.connections[pkt.ConnectionId] = route

		r.logger.Info("new connection route created",
			logger.String("conn_id", pkt.ConnectionId),
			logger.String("agent_id", agentID),
			logger.String("implant_id", implant.ID),
		)
	} else {
		route.LastActivity = time.Now()
	}

	implant, implantExists := r.implants[route.ImplantID]
	r.mu.Unlock()

	if !implantExists {
		return fmt.Errorf("implant not found: %s", route.ImplantID)
	}

	r.mu.Lock()
	route.PacketsToImplant++
	route.BytesToImplant += uint64(len(pkt.Data))
	r.mu.Unlock()

	r.logger.Debug("routing packet from agent to implant",
		logger.String("conn_id", pkt.ConnectionId),
		logger.String("implant_id", route.ImplantID),
		logger.Int("size", len(pkt.Data)),
	)

	return implant.Stream.Send(&pb.ImplantMessage{
		Message: &pb.ImplantMessage_Packet{Packet: pkt},
	})
}

func (r *Registry) RouteFromImplant(implantID string, pkt *pb.Packet) error {
	r.mu.Lock()
	route, exists := r.connections[pkt.ConnectionId]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("connection not found: %s", pkt.ConnectionId)
	}

	route.LastActivity = time.Now()
	agent, agentExists := r.agents[route.AgentID]
	r.mu.Unlock()

	if !agentExists {
		return fmt.Errorf("agent not found: %s", route.AgentID)
	}

	r.mu.Lock()
	route.PacketsToAgent++
	route.BytesToAgent += uint64(len(pkt.Data))
	r.mu.Unlock()

	r.logger.Debug("routing packet from implant to agent",
		logger.String("conn_id", pkt.ConnectionId),
		logger.String("agent_id", route.AgentID),
		logger.Int("size", len(pkt.Data)),
	)

	return agent.Stream.Send(&pb.AgentMessage{
		Message: &pb.AgentMessage_Packet{Packet: pkt},
	})
}

func (r *Registry) RemoveConnection(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.connections, connID)
	r.logger.Debug("connection route removed", logger.String("conn_id", connID))
}

func (r *Registry) GetConnectionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connections)
}

type ConnectionMetrics struct {
	ConnectionID     string
	AgentID          string
	ImplantID        string
	Age              time.Duration
	IdleTime         time.Duration
	PacketsToAgent   uint64
	PacketsToImplant uint64
	BytesToAgent     uint64
	BytesToImplant   uint64
}

func (r *Registry) GetConnectionMetrics(connID string) (*ConnectionMetrics, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, exists := r.connections[connID]
	if !exists {
		return nil, false
	}

	now := time.Now()
	return &ConnectionMetrics{
		ConnectionID:     route.ConnectionID,
		AgentID:          route.AgentID,
		ImplantID:        route.ImplantID,
		Age:              now.Sub(route.CreatedAt),
		IdleTime:         now.Sub(route.LastActivity),
		PacketsToAgent:   route.PacketsToAgent,
		PacketsToImplant: route.PacketsToImplant,
		BytesToAgent:     route.BytesToAgent,
		BytesToImplant:   route.BytesToImplant,
	}, true
}

func (r *Registry) GetAllConnectionMetrics() []*ConnectionMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]*ConnectionMetrics, 0, len(r.connections))
	now := time.Now()

	for _, route := range r.connections {
		metrics = append(metrics, &ConnectionMetrics{
			ConnectionID:     route.ConnectionID,
			AgentID:          route.AgentID,
			ImplantID:        route.ImplantID,
			Age:              now.Sub(route.CreatedAt),
			IdleTime:         now.Sub(route.LastActivity),
			PacketsToAgent:   route.PacketsToAgent,
			PacketsToImplant: route.PacketsToImplant,
			BytesToAgent:     route.BytesToAgent,
			BytesToImplant:   route.BytesToImplant,
		})
	}

	return metrics
}

func (r *Registry) findImplantByCIDR(targetIP string) (*ImplantConn, bool) {
	ip := net.ParseIP(targetIP)
	if ip == nil {
		r.logger.Warn("invalid target IP", logger.String("ip", targetIP))
		return nil, false
	}

	for _, implant := range r.implants {
		_, cidr, err := net.ParseCIDR(implant.ManagedCIDR)
		if err != nil {
			r.logger.Warn("invalid implant CIDR",
				logger.String("implant_id", implant.ID),
				logger.String("cidr", implant.ManagedCIDR),
				logger.Error(err),
			)
			continue
		}

		if cidr.Contains(ip) {
			return implant, true
		}
	}
	return nil, false
}
