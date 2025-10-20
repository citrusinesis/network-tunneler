package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"network-tunneler/pkg/logger"
	pb "network-tunneler/proto"
)

type ServerConnection struct {
	serverAddr   string
	tlsConfig    *tls.Config
	tracker      *ConnectionTracker
	logger       logger.Logger
	grpcInsecure bool
	config       *Config
	clientID     string

	grpcConn   *grpc.ClientConn
	grpcClient pb.TunnelClientClient
	stream     pb.TunnelClient_ConnectClient

	packetChan chan *pb.Packet
	stopChan   chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

type ServerConnParams struct {
	fx.In

	Config    *Config
	TLSConfig *tls.Config
	Tracker   *ConnectionTracker
	Logger    logger.Logger
}

func NewServerConnection(p ServerConnParams) *ServerConnection {
	return &ServerConnection{
		serverAddr: p.Config.ServerAddr,
		tlsConfig:  p.TLSConfig,
		tracker:    p.Tracker,
		config:     p.Config,
		packetChan: make(chan *pb.Packet, 100),
		logger:     p.Logger.With(logger.String("component", "server_conn")),
		stopChan:   make(chan struct{}),
	}
}

func (sc *ServerConnection) Connect(ctx context.Context) error {
	sc.logger.Info("connecting to server via gRPC",
		logger.String("server_addr", sc.serverAddr),
	)

	var opts []grpc.DialOption
	var creds credentials.TransportCredentials
	if sc.grpcInsecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(sc.tlsConfig)
	}
	opts = append(opts, grpc.WithTransportCredentials(creds))

	conn, err := grpc.NewClient(sc.serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	sc.grpcConn = conn
	sc.grpcClient = pb.NewTunnelClientClient(conn)

	stream, err := sc.grpcClient.Connect(ctx)
	if err != nil {
		sc.grpcConn.Close()
		return fmt.Errorf("failed to create stream: %w", err)
	}

	sc.stream = stream
	sc.logger.Info("gRPC stream established")

	if err := sc.register(); err != nil {
		sc.Close()
		return fmt.Errorf("failed to register with server: %w", err)
	}

	sc.wg.Add(2)
	go sc.readLoop()
	go sc.writeLoop()

	return nil
}

func (sc *ServerConnection) register() error {
	clientID := sc.config.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("client-%s", generateAlphanumericID(16))
		sc.clientID = clientID
	} else {
		sc.clientID = clientID
	}

	reg := &pb.ClientMessage{
		Message: &pb.ClientMessage_Register{
			Register: &pb.ClientRegister{
				ClientId: sc.clientID,
			},
		},
	}

	if err := sc.stream.Send(reg); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	sc.logger.Info("registration sent", logger.String("client_id", sc.clientID))

	msg, err := sc.stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to read registration response: %w", err)
	}

	ack, ok := msg.Message.(*pb.ClientMessage_Ack)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", msg.Message)
	}

	if !ack.Ack.Success {
		return fmt.Errorf("registration failed: %s", ack.Ack.Message)
	}

	sc.logger.Info("registered with server successfully")

	return nil
}

func (sc *ServerConnection) readLoop() {
	defer sc.wg.Done()
	defer sc.logger.Info("read loop stopped")

	for {
		select {
		case <-sc.stopChan:
			return
		default:
		}

		msg, err := sc.stream.Recv()
		if err != nil {
			if err == io.EOF {
				sc.logger.Info("server closed stream")
			} else {
				sc.logger.Error("stream recv error", logger.Error(err))
			}
			return
		}

		switch m := msg.Message.(type) {
		case *pb.ClientMessage_Packet:
			sc.handlePacket(m.Packet)
		case *pb.ClientMessage_Heartbeat:
			sc.logger.Debug("heartbeat received")
		default:
			sc.logger.Warn("unexpected message type",
				logger.String("type", fmt.Sprintf("%T", msg.Message)),
			)
		}
	}
}

func (sc *ServerConnection) writeLoop() {
	defer sc.wg.Done()
	defer sc.logger.Info("write loop stopped")

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-sc.stopChan:
			return

		case packet := <-sc.packetChan:
			msg := &pb.ClientMessage{
				Message: &pb.ClientMessage_Packet{
					Packet: packet,
				},
			}
			if err := sc.stream.Send(msg); err != nil {
				sc.logger.Error("failed to send packet",
					logger.Error(err),
					logger.String("connection_id", packet.ConnectionId),
				)
			}

		case <-heartbeatTicker.C:
			msg := &pb.ClientMessage{
				Message: &pb.ClientMessage_Heartbeat{
					Heartbeat: &pb.Heartbeat{
						SenderId:  sc.clientID,
						Timestamp: time.Now().Unix(),
					},
				},
			}
			if err := sc.stream.Send(msg); err != nil {
				sc.logger.Error("failed to send heartbeat", logger.Error(err))
			}
		}
	}
}

func (sc *ServerConnection) handlePacket(pkt *pb.Packet) {
	if err := sc.tracker.DeliverResponse(pkt.ConnectionId, pkt.Data); err != nil {
		sc.logger.Error("failed to deliver response",
			logger.Error(err),
			logger.String("connection_id", pkt.ConnectionId),
		)
	}
}

func (sc *ServerConnection) SendPacket(pkt *pb.Packet) {
	select {
	case sc.packetChan <- pkt:
	default:
		sc.logger.Warn("packet channel full, dropping packet",
			logger.String("connection_id", pkt.ConnectionId),
		)
	}
}

func generateAlphanumericID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLen := big.NewInt(int64(len(charset)))
	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			panic(fmt.Sprintf("failed to generate random number: %v", err))
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
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

func (sc *ServerConnection) GetPacketChannel() chan<- *pb.Packet {
	return sc.packetChan
}
