package server

import (
	"context"
	"testing"
	"time"

	testutil "network-tunneler/internal/testing"
	pb "network-tunneler/proto"
)

type mockAgentStream struct {
	pb.TunnelAgent_ConnectServer
}

type mockImplantStream struct {
	pb.TunnelImplant_ConnectServer
}

func TestRegistry_RegisterAgentStream(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockAgentStream{}

	err := registry.RegisterAgentStream("agent-1", stream)
	if err != nil {
		t.Fatalf("RegisterAgentStream failed: %v", err)
	}

	agent, exists := registry.GetAgent("agent-1")
	if !exists {
		t.Error("expected agent to exist")
	}

	if agent.ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %s", agent.ID)
	}

	if agent.Stream != stream {
		t.Error("expected agent stream to match")
	}
}

func TestRegistry_RegisterAgentStream_Duplicate(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockAgentStream{}

	err := registry.RegisterAgentStream("agent-1", stream)
	if err != nil {
		t.Fatalf("RegisterAgentStream failed: %v", err)
	}

	err = registry.RegisterAgentStream("agent-1", stream)
	if err == nil {
		t.Error("expected error for duplicate agent registration")
	}
}

func TestRegistry_UnregisterAgent(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockAgentStream{}

	registry.RegisterAgentStream("agent-1", stream)
	registry.UnregisterAgent("agent-1")

	_, exists := registry.GetAgent("agent-1")
	if exists {
		t.Error("expected agent to be unregistered")
	}
}

func TestRegistry_RegisterImplantStream(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockImplantStream{}

	err := registry.RegisterImplantStream("implant-1", stream, "192.168.1.0/24")
	if err != nil {
		t.Fatalf("RegisterImplantStream failed: %v", err)
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

func TestRegistry_FindImplantByCIDR(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockImplantStream{}

	registry.RegisterImplantStream("implant-1", stream, "192.168.1.0/24")

	implant, found := registry.FindImplantByCIDR("192.168.1.100")
	if !found {
		t.Fatal("expected to find implant for IP in CIDR range")
	}

	if implant.ID != "implant-1" {
		t.Errorf("expected implant-1, got %s", implant.ID)
	}

	_, found = registry.FindImplantByCIDR("10.0.0.1")
	if found {
		t.Error("expected not to find implant for IP outside CIDR range")
	}
}

func TestRegistry_Cleanup(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)

	registry.RegisterAgentStream("agent-1", &mockAgentStream{})
	registry.RegisterImplantStream("implant-1", &mockImplantStream{}, "192.168.1.0/24")

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
