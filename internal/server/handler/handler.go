package handler

import (
	"net"
	pb "network-tunneler/proto"
)

type Handler interface {
	Handle(conn net.Conn)
}

type RegistryAPI interface {
	RegisterAgent(id string, conn net.Conn) error
	UnregisterAgent(id string)
	RegisterImplant(id string, conn net.Conn, managedCIDR string) error
	UnregisterImplant(id string)
}

type RouterAPI interface {
	RouteFromAgent(pkt *pb.Packet) error
	RouteFromImplant(pkt *pb.Packet) error
}
