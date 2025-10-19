package handler

import (
	"encoding/binary"
	"io"
	"testing"

	testutil "network-tunneler/internal/testing"
	pb "network-tunneler/proto"

	"google.golang.org/protobuf/proto"
)

func TestImplant_ReadMessage_ImplantRegister(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewImplant(registry, router, log)

	conn := testutil.NewMockNetConn()

	msg := &pb.ImplantRegister{
		ImplantId:   "test-implant",
		ManagedCidr: "192.168.1.0/24",
	}

	if err := writeMessage(conn.ReadBuf, pb.MessageType_IMPLANT_REGISTER, msg); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	result, err := handler.readMessage(conn)
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	implantReg, ok := result.(*pb.ImplantRegister)
	if !ok {
		t.Fatalf("expected *pb.ImplantRegister, got %T", result)
	}

	if implantReg.ImplantId != "test-implant" {
		t.Errorf("expected implant ID 'test-implant', got %s", implantReg.ImplantId)
	}

	if implantReg.ManagedCidr != "192.168.1.0/24" {
		t.Errorf("expected managed CIDR '192.168.1.0/24', got %s", implantReg.ManagedCidr)
	}
}

func TestImplant_SendMessage(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewImplant(registry, router, log)

	conn := testutil.NewMockNetConn()

	msg := &pb.RegisterAck{
		Success: true,
		Message: "registered successfully",
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

	if ack.Message != "registered successfully" {
		t.Errorf("expected message 'registered successfully', got %s", ack.Message)
	}
}

func TestImplant_HandleMessage_Packet(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewImplant(registry, router, log)

	pkt := &pb.Packet{
		ConnectionId: "conn-2",
		Data:         []byte("response data"),
	}

	handler.handleMessage("implant-1", pkt)

	if len(router.implantPackets) != 1 {
		t.Errorf("expected 1 routed packet, got %d", len(router.implantPackets))
	}

	if router.implantPackets[0].ConnectionId != "conn-2" {
		t.Errorf("expected connection ID 'conn-2', got %s", router.implantPackets[0].ConnectionId)
	}
}

func TestImplant_HandleMessage_Heartbeat(t *testing.T) {
	log := testutil.NewTestLogger()

	registry := newMockRegistryAPI()
	router := newMockRouterAPI()
	handler := NewImplant(registry, router, log)

	hb := &pb.Heartbeat{
		SenderId:  "implant-1",
		Timestamp: 1234567890,
	}

	handler.handleMessage("implant-1", hb)

	if len(router.implantPackets) != 0 {
		t.Errorf("expected 0 routed packets for heartbeat, got %d", len(router.implantPackets))
	}
}
