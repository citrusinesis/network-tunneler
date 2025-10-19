package server

import (
	"context"
	testutil "network-tunneler/internal/testing"
	"testing"
	"time"
)

func TestRegistry_RegisterAgent(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	conn := testutil.NewMockConn()

	err := registry.RegisterAgent("agent-1", conn)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	agent, exists := registry.GetAgent("agent-1")
	if !exists {
		t.Error("expected agent to exist")
	}

	if agent.ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %s", agent.ID)
	}

	if agent.Conn != conn {
		t.Error("expected agent connection to match")
	}
}

func TestRegistry_RegisterAgent_Duplicate(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	conn := testutil.NewMockConn()

	err := registry.RegisterAgent("agent-1", conn)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	err = registry.RegisterAgent("agent-1", conn)
	if err == nil {
		t.Error("expected error for duplicate agent registration")
	}
}

func TestRegistry_UnregisterAgent(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	conn := testutil.NewMockConn()

	registry.RegisterAgent("agent-1", conn)
	registry.UnregisterAgent("agent-1")

	_, exists := registry.GetAgent("agent-1")
	if exists {
		t.Error("expected agent to be unregistered")
	}
}

func TestRegistry_RegisterImplant(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	conn := testutil.NewMockConn()

	err := registry.RegisterImplant("implant-1", conn, "192.168.1.0/24")
	if err != nil {
		t.Fatalf("RegisterImplant failed: %v", err)
	}

	implant, exists := registry.GetImplant("implant-1")
	if !exists {
		t.Error("expected implant to exist")
	}

	if implant.ID != "implant-1" {
		t.Errorf("expected implant ID 'implant-1', got %s", implant.ID)
	}

	if implant.ManagedCIDR != "192.168.1.0/24" {
		t.Errorf("expected CIDR '192.168.1.0/24', got %s", implant.ManagedCIDR)
	}
}

func TestRegistry_Cleanup(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)

	registry.RegisterAgent("agent-1", testutil.NewMockConn())
	registry.RegisterImplant("implant-1", testutil.NewMockConn(), "192.168.1.0/24")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := registry.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if len(registry.ListAgents()) != 0 {
		t.Error("expected all agents to be cleaned up")
	}

	if len(registry.ListImplants()) != 0 {
		t.Error("expected all implants to be cleaned up")
	}
}
