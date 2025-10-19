package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"network-tunneler/pkg/logger"
)

type AgentConn struct {
	ID          string
	Conn        net.Conn
	RemoteAddr  string
	ConnectedAt time.Time
}

type ImplantConn struct {
	ID          string
	Conn        net.Conn
	RemoteAddr  string
	ManagedCIDR string
	ConnectedAt time.Time
}

type Registry struct {
	agents   map[string]*AgentConn
	implants map[string]*ImplantConn
	mu       sync.RWMutex
	logger   logger.Logger
}

func NewRegistry(log logger.Logger) *Registry {
	return &Registry{
		agents:   make(map[string]*AgentConn),
		implants: make(map[string]*ImplantConn),
		logger:   log.With(logger.String("component", "registry")),
	}
}

func (r *Registry) RegisterAgent(id string, conn net.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		return fmt.Errorf("agent %s already registered", id)
	}

	agent := &AgentConn{
		ID:          id,
		Conn:        conn,
		RemoteAddr:  conn.RemoteAddr().String(),
		ConnectedAt: time.Now(),
	}

	r.agents[id] = agent
	r.logger.Info("agent registered",
		logger.String("agent_id", id),
		logger.String("remote", agent.RemoteAddr),
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

func (r *Registry) RegisterImplant(id string, conn net.Conn, managedCIDR string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.implants[id]; exists {
		return fmt.Errorf("implant %s already registered", id)
	}

	implant := &ImplantConn{
		ID:          id,
		Conn:        conn,
		RemoteAddr:  conn.RemoteAddr().String(),
		ManagedCIDR: managedCIDR,
		ConnectedAt: time.Now(),
	}

	r.implants[id] = implant
	r.logger.Info("implant registered",
		logger.String("implant_id", id),
		logger.String("remote", implant.RemoteAddr),
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

	// TODO: Implement CIDR matching logic
	for _, implant := range r.implants {
		return implant, true
	}

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
	)

	for id, agent := range r.agents {
		agent.Conn.Close()
		delete(r.agents, id)
	}

	for id, implant := range r.implants {
		implant.Conn.Close()
		delete(r.implants, id)
	}

	return nil
}
