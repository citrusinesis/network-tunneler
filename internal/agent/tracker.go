package agent

import (
	"fmt"
	"net"
	"sync"
	"time"

	"network-tunneler/pkg/logger"

	"go.uber.org/fx"
)

type ConnectionState struct {
	ConnectionID string
	OriginalDest string
	LocalConn    net.Conn
	CreatedAt    time.Time
	LastActivity time.Time
}

type ConnectionTracker struct {
	connections map[string]*ConnectionState
	mu          sync.RWMutex
	logger      logger.Logger
}

type TrackerParams struct {
	fx.In

	Logger logger.Logger
}

func NewConnectionTracker(p TrackerParams) *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]*ConnectionState),
		logger:      p.Logger.With(logger.String("component", "tracker")),
	}
}

func (ct *ConnectionTracker) Track(connID string, originalDest string, localConn net.Conn) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	ct.connections[connID] = &ConnectionState{
		ConnectionID: connID,
		OriginalDest: originalDest,
		LocalConn:    localConn,
		CreatedAt:    now,
		LastActivity: now,
	}

	ct.logger.Debug("connection tracked",
		logger.String("connection_id", connID),
		logger.String("original_dest", originalDest),
	)
}

func (ct *ConnectionTracker) Get(connID string) (*ConnectionState, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	state, exists := ct.connections[connID]
	return state, exists
}

func (ct *ConnectionTracker) UpdateActivity(connID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if state, exists := ct.connections[connID]; exists {
		state.LastActivity = time.Now()
	}
}

func (ct *ConnectionTracker) Remove(connID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if state, exists := ct.connections[connID]; exists {
		state.LocalConn.Close()
		delete(ct.connections, connID)

		ct.logger.Debug("connection removed",
			logger.String("connection_id", connID),
		)
	}
}

func (ct *ConnectionTracker) Cleanup(maxIdleTime time.Duration) int {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	removed := 0

	for connID, state := range ct.connections {
		if now.Sub(state.LastActivity) > maxIdleTime {
			state.LocalConn.Close()
			delete(ct.connections, connID)
			removed++

			ct.logger.Debug("idle connection cleaned up",
				logger.String("connection_id", connID),
				logger.Duration("idle_time", now.Sub(state.LastActivity)),
			)
		}
	}

	if removed > 0 {
		ct.logger.Info("cleanup completed",
			logger.Int("removed_connections", removed),
		)
	}

	return removed
}

func (ct *ConnectionTracker) DeliverResponse(connID string, data []byte) error {
	ct.mu.RLock()
	state, exists := ct.connections[connID]
	ct.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", connID)
	}

	ct.mu.Lock()
	state.LastActivity = time.Now()
	ct.mu.Unlock()

	_, err := state.LocalConn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to local connection: %w", err)
	}

	ct.logger.Debug("response delivered",
		logger.String("connection_id", connID),
		logger.Int("bytes", len(data)),
	)

	return nil
}

func (ct *ConnectionTracker) Count() int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return len(ct.connections)
}
