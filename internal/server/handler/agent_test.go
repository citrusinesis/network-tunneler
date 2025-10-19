package handler

import (
	"encoding/binary"
	"io"
	"testing"

	"google.golang.org/protobuf/proto"

	testutil "network-tunneler/internal/testing"
	pb "network-tunneler/proto"
)

func TestAgent_ReadMessage_AgentRegister(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewAgent(registry, router, log)

	conn := testutil.NewMockNetConn()

	msg := &pb.AgentRegister{
		AgentId: "test-agent",
	}

	if err := writeMessage(conn.ReadBuf, pb.MessageType_AGENT_REGISTER, msg); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	result, err := handler.readMessage(conn)
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	agentReg, ok := result.(*pb.AgentRegister)
	if !ok {
		t.Fatalf("expected *pb.AgentRegister, got %T", result)
	}

	if agentReg.AgentId != "test-agent" {
		t.Errorf("expected agent ID 'test-agent', got %s", agentReg.AgentId)
	}
}

func TestAgent_SendMessage(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewAgent(registry, router, log)

	conn := testutil.NewMockNetConn()

	msg := &pb.RegisterAck{
		Success: true,
		Message: "registered",
	}

	if err := handler.sendMessage(conn, msg); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var msgType uint8
	if err := binary.Read(conn.WriteBuf, binary.BigEndian, &msgType); err != nil {
		t.Fatalf("failed to read message type: %v", err)
	}

	if pb.MessageType(msgType) != pb.MessageType_REGISTER_ACK {
		t.Errorf("expected REGISTER_ACK, got %d", msgType)
	}

	var msgLen uint32
	if err := binary.Read(conn.WriteBuf, binary.BigEndian, &msgLen); err != nil {
		t.Fatalf("failed to read message length: %v", err)
	}

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn.WriteBuf, data); err != nil {
		t.Fatalf("failed to read message data: %v", err)
	}

	ack := &pb.RegisterAck{}
	if err := proto.Unmarshal(data, ack); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if !ack.Success {
		t.Error("expected success to be true")
	}

	if ack.Message != "registered" {
		t.Errorf("expected message 'registered', got %s", ack.Message)
	}
}

func TestAgent_HandleMessage_Packet(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewAgent(registry, router, log)

	pkt := &pb.Packet{
		ConnectionId: "conn-1",
		Data:         []byte("test data"),
	}

	handler.handleMessage("agent-1", pkt)

	if len(router.agentPackets) != 1 {
		t.Errorf("expected 1 routed packet, got %d", len(router.agentPackets))
	}

	if router.agentPackets[0].ConnectionId != "conn-1" {
		t.Errorf("expected connection ID 'conn-1', got %s", router.agentPackets[0].ConnectionId)
	}
}
