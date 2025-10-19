package server

import (
	"testing"

	testutil "network-tunneler/internal/testing"
	"network-tunneler/proto"
)

func TestRouter_RouteFromAgent(t *testing.T) {
	log := testutil.NewTestLogger()

	router := &Router{
		logger:      log,
		connections: make(map[string]*Connection),
	}

	agentConn := testutil.NewMockConn()
	implantConn := testutil.NewMockConn()

	router.connections["conn-1"] = &Connection{
		AgentConn:   agentConn,
		ImplantConn: implantConn,
		ConnID:      "conn-1",
	}

	pkt := &proto.Packet{
		ConnectionId: "conn-1",
		Data:         []byte("test data"),
	}

	err := router.RouteFromAgent(pkt)
	if err != nil {
		t.Errorf("RouteFromAgent failed: %v", err)
	}
}

func TestRouter_RouteFromAgent_NoConnection(t *testing.T) {
	log := testutil.NewTestLogger()

	router := &Router{
		logger:      log,
		connections: make(map[string]*Connection),
	}

	pkt := &proto.Packet{
		ConnectionId: "nonexistent",
		Data:         []byte("test data"),
	}

	err := router.RouteFromAgent(pkt)
	if err == nil {
		t.Error("expected error for nonexistent connection")
	}
}

func TestRouter_RouteFromImplant(t *testing.T) {
	log := testutil.NewTestLogger()

	router := &Router{
		logger:      log,
		connections: make(map[string]*Connection),
	}

	agentConn := testutil.NewMockConn()
	implantConn := testutil.NewMockConn()

	router.connections["conn-1"] = &Connection{
		AgentConn:   agentConn,
		ImplantConn: implantConn,
		ConnID:      "conn-1",
	}

	pkt := &proto.Packet{
		ConnectionId: "conn-1",
		Data:         []byte("test data"),
	}

	err := router.RouteFromImplant(pkt)
	if err != nil {
		t.Errorf("RouteFromImplant failed: %v", err)
	}
}

func TestRouter_RegisterAgent(t *testing.T) {
	log := testutil.NewTestLogger()

	router := &Router{
		logger:      log,
		connections: make(map[string]*Connection),
	}

	agentConn := testutil.NewMockConn()

	err := router.RegisterAgent("conn-1", agentConn)
	if err != nil {
		t.Errorf("RegisterAgent failed: %v", err)
	}

	conn, exists := router.connections["conn-1"]
	if !exists {
		t.Error("expected connection to exist")
	}

	if conn.AgentConn != agentConn {
		t.Error("expected agent connection to match")
	}
}
