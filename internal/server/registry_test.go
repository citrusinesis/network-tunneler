package server

import (
	"context"
	"testing"
	"time"

	testutil "network-tunneler/internal/testing"
	pb "network-tunneler/proto"
)

type mockClientStream struct {
	pb.TunnelClient_ConnectServer
}

type mockProxyStream struct {
	pb.TunnelProxy_ConnectServer
}

func TestRegistry_RegisterClientStream(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockClientStream{}

	err := registry.RegisterClientStream("client-1", stream)
	if err != nil {
		t.Fatalf("RegisterClientStream failed: %v", err)
	}

	client, exists := registry.GetClient("client-1")
	if !exists {
		t.Error("expected client to exist")
	}

	if client.ID != "client-1" {
		t.Errorf("expected client ID 'client-1', got %s", client.ID)
	}

	if client.Stream != stream {
		t.Error("expected client stream to match")
	}
}

func TestRegistry_RegisterClientStream_Duplicate(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockClientStream{}

	err := registry.RegisterClientStream("client-1", stream)
	if err != nil {
		t.Fatalf("RegisterClientStream failed: %v", err)
	}

	err = registry.RegisterClientStream("client-1", stream)
	if err == nil {
		t.Error("expected error for duplicate client registration")
	}
}

func TestRegistry_UnregisterClient(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockClientStream{}

	registry.RegisterClientStream("client-1", stream)
	registry.UnregisterClient("client-1")

	_, exists := registry.GetClient("client-1")
	if exists {
		t.Error("expected client to be unregistered")
	}
}

func TestRegistry_RegisterProxyStream(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockProxyStream{}

	err := registry.RegisterProxyStream("proxy-1", stream, "192.168.1.0/24")
	if err != nil {
		t.Fatalf("RegisterProxyStream failed: %v", err)
	}

	proxy, exists := registry.GetProxy("proxy-1")
	if !exists {
		t.Error("expected proxy to exist")
	}

	if proxy.ID != "proxy-1" {
		t.Errorf("expected proxy ID 'proxy-1', got %s", proxy.ID)
	}

	if proxy.ManagedCIDR != "192.168.1.0/24" {
		t.Errorf("expected CIDR '192.168.1.0/24', got %s", proxy.ManagedCIDR)
	}
}

func TestRegistry_FindProxyByCIDR(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)
	stream := &mockProxyStream{}

	registry.RegisterProxyStream("proxy-1", stream, "192.168.1.0/24")

	proxy, found := registry.FindProxyByCIDR("192.168.1.100")
	if !found {
		t.Fatal("expected to find proxy for IP in CIDR range")
	}

	if proxy.ID != "proxy-1" {
		t.Errorf("expected proxy-1, got %s", proxy.ID)
	}

	_, found = registry.FindProxyByCIDR("10.0.0.1")
	if found {
		t.Error("expected not to find proxy for IP outside CIDR range")
	}
}

func TestRegistry_Cleanup(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := NewRegistry(log)

	registry.RegisterClientStream("client-1", &mockClientStream{})
	registry.RegisterProxyStream("proxy-1", &mockProxyStream{}, "192.168.1.0/24")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := registry.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if len(registry.ListClients()) != 0 {
		t.Error("expected all clients to be cleaned up")
	}

	if len(registry.ListProxys()) != 0 {
		t.Error("expected all proxys to be cleaned up")
	}
}
