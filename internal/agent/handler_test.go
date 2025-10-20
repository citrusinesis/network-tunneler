package agent

import (
	"net"
	"testing"
	"time"

	testutil "network-tunneler/internal/testing"
	pb "network-tunneler/proto"
)

func TestConnectionHandler_Handle(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	serverChan := make(chan *pb.Packet, 10)
	handler := NewConnectionHandler(tracker, serverChan, log)
	handler.getOriginalDest = func(conn net.Conn) (string, error) {
		return "100.64.1.5:80", nil
	}

	mockConn := testutil.NewMockNetConn()
	mockConn.LocalAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
	mockConn.RemoteAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}

	testData := []byte("test data from client")
	mockConn.ReadBuf.Write(testData)

	done := make(chan struct{})
	go func() {
		handler.Handle(mockConn)
		close(done)
	}()

	select {
	case pkt := <-serverChan:
		if pkt.ConnectionId == "" {
			t.Error("expected non-empty connection ID")
		}
		if string(pkt.Data) != string(testData) {
			t.Errorf("expected data %s, got %s", testData, pkt.Data)
		}
		if pkt.Protocol != pb.Protocol_PROTOCOL_TCP {
			t.Errorf("expected TCP protocol, got %v", pkt.Protocol)
		}
		if pkt.Direction != pb.Direction_DIRECTION_FORWARD {
			t.Errorf("expected forward direction, got %v", pkt.Direction)
		}
		if pkt.ConnTuple == nil {
			t.Error("expected connection tuple to be set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for packet")
	}

	mockConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler to finish")
	}
}

func TestConnectionHandler_MultiplePackets(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	serverChan := make(chan *pb.Packet, 10)
	handler := NewConnectionHandler(tracker, serverChan, log)
	handler.getOriginalDest = func(conn net.Conn) (string, error) {
		return "100.64.1.5:80", nil
	}

	mockConn := testutil.NewMockNetConn()
	mockConn.LocalAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
	mockConn.RemoteAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}

	testData := []byte("combined data from multiple application writes")
	mockConn.ReadBuf.Write(testData)

	done := make(chan struct{})
	go func() {
		handler.Handle(mockConn)
		close(done)
	}()

	select {
	case pkt := <-serverChan:
		if pkt.ConnectionId == "" {
			t.Error("expected non-empty connection ID")
		}
		if pkt.ConnTuple == nil {
			t.Error("expected connection tuple to be set")
		}
		if len(pkt.Data) != len(testData) {
			t.Errorf("expected %d bytes, got %d", len(testData), len(pkt.Data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for packet")
	}

	mockConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler to finish")
	}
}

func TestConnectionHandler_ChannelFull(t *testing.T) {
	log := testutil.NewTestLogger()
	tracker := NewConnectionTracker(TrackerParams{Logger: log})

	serverChan := make(chan *pb.Packet, 1)
	handler := NewConnectionHandler(tracker, serverChan, log)
	handler.getOriginalDest = func(conn net.Conn) (string, error) {
		return "100.64.1.5:80", nil
	}

	mockConn := testutil.NewMockNetConn()
	mockConn.LocalAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
	mockConn.RemoteAddress = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}

	testData1 := []byte("first packet")
	testData2 := []byte("second packet")

	mockConn.ReadBuf.Write(testData1)
	mockConn.ReadBuf.Write(testData2)

	done := make(chan struct{})
	go func() {
		handler.Handle(mockConn)
		close(done)
	}()

	select {
	case <-serverChan:
		t.Log("First packet received")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first packet")
	}

	time.Sleep(200 * time.Millisecond)

	select {
	case <-serverChan:
		t.Log("Second packet received (channel was cleared)")
	case <-time.After(100 * time.Millisecond):
		t.Log("Second packet dropped (expected when channel is full)")
	}

	mockConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler to finish")
	}
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		wantHost    string
		wantPort    uint16
		expectError bool
	}{
		{
			name:        "valid IPv4 address",
			addr:        "192.168.1.1:8080",
			wantHost:    "192.168.1.1",
			wantPort:    8080,
			expectError: false,
		},
		{
			name:        "valid IPv6 address",
			addr:        "[::1]:8080",
			wantHost:    "::1",
			wantPort:    8080,
			expectError: false,
		},
		{
			name:        "localhost",
			addr:        "localhost:3000",
			wantHost:    "localhost",
			wantPort:    3000,
			expectError: false,
		},
		{
			name:        "missing port",
			addr:        "192.168.1.1",
			expectError: true,
		},
		{
			name:        "invalid port",
			addr:        "192.168.1.1:invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseAddr(tt.addr)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if host != tt.wantHost {
				t.Errorf("expected host %s, got %s", tt.wantHost, host)
			}

			if port != tt.wantPort {
				t.Errorf("expected port %d, got %d", tt.wantPort, port)
			}
		})
	}
}
