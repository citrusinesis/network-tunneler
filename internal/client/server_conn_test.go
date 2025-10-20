package client

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	testutil "network-tunneler/internal/testing"
	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type mockClientServer struct {
	pb.UnimplementedTunnelClientServer
	registerChan chan *pb.ClientRegister
	packetChan   chan *pb.Packet
	stream       pb.TunnelClient_ConnectServer
}

func (m *mockClientServer) Connect(stream pb.TunnelClient_ConnectServer) error {
	m.stream = stream

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		switch msg := msg.Message.(type) {
		case *pb.ClientMessage_Register:
			m.registerChan <- msg.Register
			ack := &pb.ClientMessage{
				Message: &pb.ClientMessage_Ack{
					Ack: &pb.RegisterAck{
						Success: true,
						Message: "registered successfully",
					},
				},
			}
			if err := stream.Send(ack); err != nil {
				return err
			}

		case *pb.ClientMessage_Packet:
			m.packetChan <- msg.Packet
		}
	}
}

func setupMockServer(t *testing.T) (*grpc.Server, string, *mockClientServer) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()

	mock := &mockClientServer{
		registerChan: make(chan *pb.ClientRegister, 1),
		packetChan:   make(chan *pb.Packet, 10),
	}

	pb.RegisterTunnelClientServer(server, mock)

	go server.Serve(lis)

	return server, lis.Addr().String(), mock
}

func newTestServerConnection(addr string, tracker *ConnectionTracker, log logger.Logger) *ServerConnection {
	cfg := &Config{
		ClientID:   "",
		ServerAddr: addr,
	}
	return &ServerConnection{
		serverAddr:   addr,
		tracker:      tracker,
		config:       cfg,
		packetChan:   make(chan *pb.Packet, 100),
		logger:       log.With(logger.String("component", "server_conn")),
		stopChan:     make(chan struct{}),
		grpcInsecure: true,
	}
}

func TestServerConnection_Connect(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	sc := newTestServerConnection(addr, tracker, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sc.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	select {
	case reg := <-mock.registerChan:
		if !strings.HasPrefix(reg.ClientId, "client-") {
			t.Errorf("expected client_id to have prefix 'client-', got %s", reg.ClientId)
		}
		if len(reg.ClientId) != 23 {
			t.Errorf("expected client_id length 23, got %d", len(reg.ClientId))
		}
		idPart := reg.ClientId[7:]
		for _, ch := range idPart {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				t.Errorf("expected client_id to contain only alphanumeric characters, got %s", reg.ClientId)
				break
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for registration")
	}
}

func TestServerConnection_SendPacket(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	sc := newTestServerConnection(addr, tracker, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	<-mock.registerChan

	testPacket := &pb.Packet{
		ConnectionId: "test-conn-1",
		Data:         []byte("test data"),
	}

	sc.SendPacket(testPacket)

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

func TestServerConnection_ReceivePacket(t *testing.T) {
	server, addr, mock := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	sc := newTestServerConnection(addr, tracker, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	<-mock.registerChan

	connID := "test-conn-1"

	mockConn := testutil.NewMockNetConn()
	tracker.Track(connID, "192.168.1.1:80", mockConn)

	responseData := []byte("response data")
	responsePacket := &pb.ClientMessage{
		Message: &pb.ClientMessage_Packet{
			Packet: &pb.Packet{
				ConnectionId: connID,
				Data:         responseData,
			},
		},
	}

	if err := mock.stream.Send(responsePacket); err != nil {
		t.Fatalf("failed to send response: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	receivedData := mockConn.WriteBuf.Bytes()

	if string(receivedData) != string(responseData) {
		t.Errorf("expected data %s, got %s", responseData, receivedData)
	}
}

func TestServerConnection_Close(t *testing.T) {
	server, addr, _ := setupMockServer(t)
	defer server.Stop()

	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	sc := newTestServerConnection(addr, tracker, log)

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
	sc.SendPacket(testPacket)
}
