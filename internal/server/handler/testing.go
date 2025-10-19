package handler

import (
	"bytes"
	"encoding/binary"
	"net"

	pb "network-tunneler/proto"

	"google.golang.org/protobuf/proto"
)

type mockRegistryAPI struct {
	agents   map[string]net.Conn
	implants map[string]net.Conn
}

func newMockRegistryAPI() *mockRegistryAPI {
	return &mockRegistryAPI{
		agents:   make(map[string]net.Conn),
		implants: make(map[string]net.Conn),
	}
}

func (m *mockRegistryAPI) RegisterAgent(id string, conn net.Conn) error {
	m.agents[id] = conn
	return nil
}

func (m *mockRegistryAPI) UnregisterAgent(id string) {
	delete(m.agents, id)
}

func (m *mockRegistryAPI) RegisterImplant(id string, conn net.Conn, managedCIDR string) error {
	m.implants[id] = conn
	return nil
}

func (m *mockRegistryAPI) UnregisterImplant(id string) {
	delete(m.implants, id)
}

type mockRouterAPI struct {
	agentPackets   []*pb.Packet
	implantPackets []*pb.Packet
}

func newMockRouterAPI() *mockRouterAPI {
	return &mockRouterAPI{
		agentPackets:   make([]*pb.Packet, 0),
		implantPackets: make([]*pb.Packet, 0),
	}
}

func (m *mockRouterAPI) RouteFromAgent(pkt *pb.Packet) error {
	m.agentPackets = append(m.agentPackets, pkt)
	return nil
}

func (m *mockRouterAPI) RouteFromImplant(pkt *pb.Packet) error {
	m.implantPackets = append(m.implantPackets, pkt)
	return nil
}

func writeMessage(buf *bytes.Buffer, msgType pb.MessageType, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, uint8(msgType)); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = buf.Write(data)
	return err
}
