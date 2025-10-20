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

type ClientConn struct {
	ID          string
	Stream      pb.TunnelClient_ConnectServer
	RemoteAddr  string
	ConnectedAt time.Time
}

type ProxyConn struct {
	ID          string
	Stream      pb.TunnelProxy_ConnectServer
	RemoteAddr  string
	ManagedCIDR string
	ConnectedAt time.Time
}

type Registry struct {
	clients      map[string]*ClientConn
	proxys    map[string]*ProxyConn
	connections map[string]*ConnectionRoute // connectionID -> route
	mu          sync.RWMutex
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

type ConnectionRoute struct {
	ConnectionID     string
	ClientID          string
	ProxyID        string
	CreatedAt        time.Time
	LastActivity     time.Time
	PacketsToClient   uint64
	PacketsToProxy uint64
	BytesToClient     uint64
	BytesToProxy   uint64
}

func NewRegistry(log logger.Logger) *Registry {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Registry{
		clients:      make(map[string]*ClientConn),
		proxys:    make(map[string]*ProxyConn),
		connections: make(map[string]*ConnectionRoute),
		logger:      log.With(logger.String("component", "registry")),
		ctx:         ctx,
		cancel:      cancel,
	}

	r.wg.Add(1)
	go r.cleanupLoop()

	return r
}

func (r *Registry) cleanupLoop() {
	defer r.wg.Done()
	defer r.logger.Info("cleanup loop stopped")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanupStaleConnections()
		case <-r.ctx.Done():
			r.logger.Debug("context cancelled, stopping cleanup loop")
			return
		}
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

func (r *Registry) RegisterClientStream(id string, stream pb.TunnelClient_ConnectServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[id]; exists {
		return fmt.Errorf("client %s already registered", id)
	}

	client := &ClientConn{
		ID:          id,
		Stream:      stream,
		RemoteAddr:  "grpc-stream",
		ConnectedAt: time.Now(),
	}

	r.clients[id] = client
	r.logger.Info("client registered via gRPC",
		logger.String("client_id", id),
	)

	return nil
}

func (r *Registry) UnregisterClient(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if client, exists := r.clients[id]; exists {
		delete(r.clients, id)
		r.logger.Info("client unregistered",
			logger.String("client_id", id),
			logger.String("remote", client.RemoteAddr),
		)
	}
}

func (r *Registry) GetClient(id string) (*ClientConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clients[id]
	return client, exists
}

func (r *Registry) RegisterProxyStream(id string, stream pb.TunnelProxy_ConnectServer, managedCIDR string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.proxys[id]; exists {
		return fmt.Errorf("proxy %s already registered", id)
	}

	proxy := &ProxyConn{
		ID:          id,
		Stream:      stream,
		RemoteAddr:  "grpc-stream",
		ManagedCIDR: managedCIDR,
		ConnectedAt: time.Now(),
	}

	r.proxys[id] = proxy
	r.logger.Info("proxy registered via gRPC",
		logger.String("proxy_id", id),
		logger.String("managed_cidr", managedCIDR),
	)

	return nil
}

func (r *Registry) UnregisterProxy(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if proxy, exists := r.proxys[id]; exists {
		delete(r.proxys, id)
		r.logger.Info("proxy unregistered",
			logger.String("proxy_id", id),
			logger.String("remote", proxy.RemoteAddr),
		)
	}
}

func (r *Registry) GetProxy(id string) (*ProxyConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proxy, exists := r.proxys[id]
	return proxy, exists
}

func (r *Registry) FindProxyByCIDR(targetIP string) (*ProxyConn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ip := net.ParseIP(targetIP)
	if ip == nil {
		r.logger.Warn("invalid target IP", logger.String("ip", targetIP))
		return nil, false
	}

	for _, proxy := range r.proxys {
		_, cidr, err := net.ParseCIDR(proxy.ManagedCIDR)
		if err != nil {
			r.logger.Warn("invalid proxy CIDR",
				logger.String("proxy_id", proxy.ID),
				logger.String("cidr", proxy.ManagedCIDR),
				logger.Error(err),
			)
			continue
		}

		if cidr.Contains(ip) {
			r.logger.Debug("found proxy for target IP",
				logger.String("target_ip", targetIP),
				logger.String("proxy_id", proxy.ID),
				logger.String("managed_cidr", proxy.ManagedCIDR),
			)
			return proxy, true
		}
	}

	r.logger.Warn("no proxy found for target IP", logger.String("ip", targetIP))
	return nil, false
}

func (r *Registry) ListClients() []*ClientConn {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := make([]*ClientConn, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}

func (r *Registry) ListProxys() []*ProxyConn {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proxys := make([]*ProxyConn, 0, len(r.proxys))
	for _, proxy := range r.proxys {
		proxys = append(proxys, proxy)
	}
	return proxys
}

func (r *Registry) Cleanup(ctx context.Context) error {
	r.logger.Info("cleaning up registry",
		logger.Int("clients", len(r.clients)),
		logger.Int("proxys", len(r.proxys)),
		logger.Int("connections", len(r.connections)),
	)

	r.cancel()
	r.wg.Wait()

	r.mu.Lock()
	r.clients = make(map[string]*ClientConn)
	r.proxys = make(map[string]*ProxyConn)
	r.connections = make(map[string]*ConnectionRoute)
	r.mu.Unlock()

	r.logger.Info("registry cleaned up")

	return nil
}

func (r *Registry) RouteFromClient(clientID string, pkt *pb.Packet) error {
	r.mu.Lock()

	route, exists := r.connections[pkt.ConnectionId]
	if !exists {
		destIP := ""
		if pkt.ConnTuple != nil {
			destIP = pkt.ConnTuple.DstIp
		}
		proxy, found := r.findProxyByCIDR(destIP)
		if !found {
			r.mu.Unlock()
			return fmt.Errorf("no proxy found for destination: %s", destIP)
		}

		now := time.Now()
		route = &ConnectionRoute{
			ConnectionID: pkt.ConnectionId,
			ClientID:      clientID,
			ProxyID:    proxy.ID,
			CreatedAt:    now,
			LastActivity: now,
		}
		r.connections[pkt.ConnectionId] = route

		r.logger.Info("new connection route created",
			logger.String("conn_id", pkt.ConnectionId),
			logger.String("client_id", clientID),
			logger.String("proxy_id", proxy.ID),
		)
	} else {
		route.LastActivity = time.Now()
	}

	proxy, proxyExists := r.proxys[route.ProxyID]
	r.mu.Unlock()

	if !proxyExists {
		return fmt.Errorf("proxy not found: %s", route.ProxyID)
	}

	r.mu.Lock()
	route.PacketsToProxy++
	route.BytesToProxy += uint64(len(pkt.Data))
	r.mu.Unlock()

	r.logger.Debug("routing packet from client to proxy",
		logger.String("conn_id", pkt.ConnectionId),
		logger.String("proxy_id", route.ProxyID),
		logger.Int("size", len(pkt.Data)),
	)

	return proxy.Stream.Send(&pb.ProxyMessage{
		Message: &pb.ProxyMessage_Packet{Packet: pkt},
	})
}

func (r *Registry) RouteFromProxy(proxyID string, pkt *pb.Packet) error {
	r.mu.Lock()
	route, exists := r.connections[pkt.ConnectionId]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("connection not found: %s", pkt.ConnectionId)
	}

	route.LastActivity = time.Now()
	client, clientExists := r.clients[route.ClientID]
	r.mu.Unlock()

	if !clientExists {
		return fmt.Errorf("client not found: %s", route.ClientID)
	}

	r.mu.Lock()
	route.PacketsToClient++
	route.BytesToClient += uint64(len(pkt.Data))
	r.mu.Unlock()

	r.logger.Debug("routing packet from proxy to client",
		logger.String("conn_id", pkt.ConnectionId),
		logger.String("client_id", route.ClientID),
		logger.Int("size", len(pkt.Data)),
	)

	return client.Stream.Send(&pb.ClientMessage{
		Message: &pb.ClientMessage_Packet{Packet: pkt},
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
	ClientID          string
	ProxyID        string
	Age              time.Duration
	IdleTime         time.Duration
	PacketsToClient   uint64
	PacketsToProxy uint64
	BytesToClient     uint64
	BytesToProxy   uint64
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
		ClientID:          route.ClientID,
		ProxyID:        route.ProxyID,
		Age:              now.Sub(route.CreatedAt),
		IdleTime:         now.Sub(route.LastActivity),
		PacketsToClient:   route.PacketsToClient,
		PacketsToProxy: route.PacketsToProxy,
		BytesToClient:     route.BytesToClient,
		BytesToProxy:   route.BytesToProxy,
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
			ClientID:          route.ClientID,
			ProxyID:        route.ProxyID,
			Age:              now.Sub(route.CreatedAt),
			IdleTime:         now.Sub(route.LastActivity),
			PacketsToClient:   route.PacketsToClient,
			PacketsToProxy: route.PacketsToProxy,
			BytesToClient:     route.BytesToClient,
			BytesToProxy:   route.BytesToProxy,
		})
	}

	return metrics
}

func (r *Registry) findProxyByCIDR(targetIP string) (*ProxyConn, bool) {
	ip := net.ParseIP(targetIP)
	if ip == nil {
		r.logger.Warn("invalid target IP", logger.String("ip", targetIP))
		return nil, false
	}

	for _, proxy := range r.proxys {
		_, cidr, err := net.ParseCIDR(proxy.ManagedCIDR)
		if err != nil {
			r.logger.Warn("invalid proxy CIDR",
				logger.String("proxy_id", proxy.ID),
				logger.String("cidr", proxy.ManagedCIDR),
				logger.Error(err),
			)
			continue
		}

		if cidr.Contains(ip) {
			return proxy, true
		}
	}
	return nil, false
}
