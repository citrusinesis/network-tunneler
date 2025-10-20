package proxy

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	testutil "network-tunneler/internal/testing"
	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type mockProxyServer struct {
	pb.UnimplementedTunnelProxyServer
	registerChan  chan *pb.ProxyRegister
	packetChan    chan *pb.Packet
	heartbeatChan chan *pb.Heartbeat
	stream        pb.TunnelProxy_ConnectServer
}

func (m *mockProxyServer) Connect(stream pb.TunnelProxy_ConnectServer) error {
	m.stream = stream

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		switch msg := msg.Message.(type) {
		case *pb.ProxyMessage_Register:
			m.registerChan <- msg.Register
			ack := &pb.ProxyMessage{
				Message: &pb.ProxyMessage_Ack{
					Ack: &pb.RegisterAck{
						Success: true,
						Message: "registered successfully",
					},
				},
			}
			if err := stream.Send(ack); err != nil {
				return err
			}

		case *pb.ProxyMessage_Packet:
			m.packetChan <- msg.Packet

		case *pb.ProxyMessage_Heartbeat:
			m.heartbeatChan <- msg.Heartbeat
		}
	}
}

func setupMockServer(t *testing.T) (*grpc.Server, string, *mockProxyServer) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()

	mock := &mockProxyServer{
		registerChan:  make(chan *pb.ProxyRegister, 1),
		packetChan:    make(chan *pb.Packet, 10),
		heartbeatChan: make(chan *pb.Heartbeat, 10),
	}

	pb.RegisterTunnelProxyServer(server, mock)

	go server.Serve(lis)

	return server, lis.Addr().String(), mock
}

func newTestServerConnection(addr string, forwarder *PacketForwarder, responseChan <-chan *pb.Packet, log logger.Logger) *ServerConnection {
	return &ServerConnection{
		serverAddr:   addr,
		proxyID:    "proxy-1",
		managedCIDR:  "192.168.1.0/24",
		forwarder:    forwarder,
		logger:       log.With(logger.String("component", "server_conn")),
		responseChan: responseChan,
		stopChan:     make(chan struct{}),
		grpcInsecure: true,
	}
}

func TestServerConnection_Connect(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	responseChan := make(chan *pb.Packet, 100)
	forwarder := NewPacketForwarder(ForwarderParams{
		Logger:       log,
		ResponseChan: responseChan,
	})

	sc := newTestServerConnection(addr, forwarder, responseChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sc.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	select {
	case reg := <-mock.registerChan:
		if reg.ProxyId != "proxy-1" {
			t.Errorf("expected proxy_id 'proxy-1', got %s", reg.ProxyId)
		}
		if reg.ManagedCidr != "192.168.1.0/24" {
			t.Errorf("expected managed_cidr '192.168.1.0/24', got %s", reg.ManagedCidr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for registration")
	}
}

func TestServerConnection_SendPacket(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	responseChan := make(chan *pb.Packet, 100)
	forwarder := NewPacketForwarder(ForwarderParams{
		Logger:       log,
		ResponseChan: responseChan,
	})

	sc := newTestServerConnection(addr, forwarder, responseChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	<-mock.registerChan

	testPacket := &pb.Packet{
		ConnectionId: "test-conn-1",
		Data:         []byte("response data"),
	}

	responseChan <- testPacket

	select {
	case pkt := <-mock.packetChan:
		if pkt.ConnectionId != testPacket.ConnectionId {
			t.Errorf("expected connection_id %s, got %s", testPacket.ConnectionId, pkt.ConnectionId)
		}
		if string(pkt.Data) != string(testPacket.Data) {
			t.Errorf("expected data %s, got %s", testPacket.Data, pkt.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for packet")
	}
}

func TestServerConnection_SendHeartbeat(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	responseChan := make(chan *pb.Packet, 100)
	forwarder := NewPacketForwarder(ForwarderParams{
		Logger:       log,
		ResponseChan: responseChan,
	})

	sc := newTestServerConnection(addr, forwarder, responseChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	<-mock.registerChan

	if err := sc.SendHeartbeat(); err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}

	select {
	case hb := <-mock.heartbeatChan:
		if hb.Timestamp == 0 {
			t.Error("expected non-zero timestamp")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for heartbeat")
	}
}

func TestServerConnection_ReceivePacket(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	responseChan := make(chan *pb.Packet, 100)
	forwarder := NewPacketForwarder(ForwarderParams{
		Logger:       log,
		ResponseChan: responseChan,
	})

	sc := newTestServerConnection(addr, forwarder, responseChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	<-mock.registerChan

	incomingPacket := &pb.ProxyMessage{
		Message: &pb.ProxyMessage_Packet{
			Packet: &pb.Packet{
				ConnectionId: "test-conn-1",
				Data:         []byte("data from server"),
				ConnTuple: &pb.ConnectionTuple{
					SrcIp:   "10.0.0.1",
					SrcPort: 54321,
					DstIp:   "192.168.1.5",
					DstPort: 80,
				},
			},
		},
	}

	if err := mock.stream.Send(incomingPacket); err != nil {
		t.Fatalf("failed to send packet: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestServerConnection_Close(t *testing.T) {
	server, addr, _ := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	responseChan := make(chan *pb.Packet, 100)
	forwarder := NewPacketForwarder(ForwarderParams{
		Logger:       log,
		ResponseChan: responseChan,
	})

	sc := newTestServerConnection(addr, forwarder, responseChan, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := sc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	testPacket := &pb.Packet{
		ConnectionId: "test-conn",
		Data:         []byte("should not send"),
	}

	select {
	case responseChan <- testPacket:
		t.Log("Packet sent after close (channel still open, expected)")
	default:
		t.Log("Cannot send packet after close")
	}
}
