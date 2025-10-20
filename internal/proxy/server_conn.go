package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ServerConnection struct {
	serverAddr   string
	proxyID    string
	managedCIDR  string
	tlsConfig    *tls.Config
	logger       logger.Logger
	forwarder    *PacketForwarder
	grpcInsecure bool

	grpcConn   *grpc.ClientConn
	grpcClient pb.TunnelProxyClient
	stream     pb.TunnelProxy_ConnectClient

	responseChan <-chan *pb.Packet
	stopChan     chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

type ServerConnParams struct {
	Config       *Config
	TLSConfig    *tls.Config
	Forwarder    *PacketForwarder
	Logger       logger.Logger
	ResponseChan <-chan *pb.Packet
}

func NewServerConnection(p ServerConnParams) *ServerConnection {
	return &ServerConnection{
		serverAddr:   p.Config.ServerAddr,
		proxyID:    p.Config.ProxyID,
		managedCIDR:  p.Config.ManagedCIDR,
		tlsConfig:    p.TLSConfig,
		forwarder:    p.Forwarder,
		logger:       p.Logger.With(logger.String("component", "server_conn")),
		responseChan: p.ResponseChan,
		stopChan:     make(chan struct{}),
	}
}

func (sc *ServerConnection) Connect(ctx context.Context) error {
	sc.logger.Info("connecting to server via gRPC",
		logger.String("server_addr", sc.serverAddr),
	)

	var opts []grpc.DialOption
	if sc.grpcInsecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		creds := credentials.NewTLS(sc.tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.NewClient(sc.serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	sc.grpcConn = conn
	sc.grpcClient = pb.NewTunnelProxyClient(conn)

	stream, err := sc.grpcClient.Connect(ctx)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create stream: %w", err)
	}

	sc.stream = stream

	sc.logger.Info("gRPC stream established")

	if err := sc.register(); err != nil {
		conn.Close()
		return fmt.Errorf("registration failed: %w", err)
	}

	sc.wg.Add(2)
	go sc.readLoop()
	go sc.writeLoop()

	return nil
}

func (sc *ServerConnection) register() error {
	regMsg := &pb.ProxyMessage{
		Message: &pb.ProxyMessage_Register{
			Register: &pb.ProxyRegister{
				ProxyId:   sc.proxyID,
				ManagedCidr: sc.managedCIDR,
			},
		},
	}

	if err := sc.stream.Send(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	sc.logger.Info("registration sent",
		logger.String("proxy_id", sc.proxyID),
		logger.String("managed_cidr", sc.managedCIDR),
	)

	ackMsg, err := sc.stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive ack: %w", err)
	}

	ack, ok := ackMsg.Message.(*pb.ProxyMessage_Ack)
	if !ok {
		return fmt.Errorf("expected ack message, got %T", ackMsg.Message)
	}

	if !ack.Ack.Success {
		return fmt.Errorf("registration rejected: %s", ack.Ack.Message)
	}

	sc.logger.Info("registered with server successfully")

	return nil
}

func (sc *ServerConnection) readLoop() {
	defer sc.wg.Done()
	defer sc.logger.Info("read loop stopped")

	for {
		msg, err := sc.stream.Recv()
		if err == io.EOF {
			sc.logger.Info("stream closed by server")
			sc.stopOnce.Do(func() { close(sc.stopChan) })
			return
		}
		if err != nil {
			sc.logger.Error("stream recv error", logger.Error(err))
			sc.stopOnce.Do(func() { close(sc.stopChan) })
			return
		}

		switch m := msg.Message.(type) {
		case *pb.ProxyMessage_Packet:
			sc.logger.Debug("received packet from server",
				logger.String("conn_id", m.Packet.ConnectionId),
				logger.Int("bytes", len(m.Packet.Data)),
			)

			if err := sc.forwarder.Forward(m.Packet); err != nil {
				sc.logger.Error("failed to forward packet",
					logger.String("conn_id", m.Packet.ConnectionId),
					logger.Error(err),
				)
			}

		default:
			sc.logger.Warn("unknown message type from server")
		}
	}
}

func (sc *ServerConnection) writeLoop() {
	defer sc.wg.Done()
	defer sc.logger.Info("write loop stopped")

	for {
		select {
		case pkt := <-sc.responseChan:
			msg := &pb.ProxyMessage{
				Message: &pb.ProxyMessage_Packet{
					Packet: pkt,
				},
			}

			if err := sc.stream.Send(msg); err != nil {
				sc.logger.Error("failed to send packet",
					logger.String("conn_id", pkt.ConnectionId),
					logger.Error(err),
				)
			}

		case <-sc.stopChan:
			return
		}
	}
}

func (sc *ServerConnection) SendHeartbeat() error {
	msg := &pb.ProxyMessage{
		Message: &pb.ProxyMessage_Heartbeat{
			Heartbeat: &pb.Heartbeat{
				Timestamp: time.Now().Unix(),
			},
		},
	}

	if err := sc.stream.Send(msg); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	sc.logger.Debug("heartbeat sent")
	return nil
}

func (sc *ServerConnection) Close() error {
	sc.stopOnce.Do(func() { close(sc.stopChan) })

	if sc.stream != nil {
		sc.stream.CloseSend()
	}

	sc.wg.Wait()

	if sc.grpcConn != nil {
		return sc.grpcConn.Close()
	}

	return nil
}
