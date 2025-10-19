package testing

import (
	"bytes"
	"net"
	"time"

	"network-tunneler/pkg/logger"
)

func NewTestLogger() logger.Logger {
	log, _ := logger.NewSlogLogger(&logger.Config{
		Level:       logger.LevelDebug,
		Format:      logger.FormatConsole,
		Development: true,
	})
	return log
}

func NewMockConn() net.Conn {
	client, _ := net.Pipe()
	return client
}

func NewMockConnPair() (client, server net.Conn) {
	return net.Pipe()
}

type MockNetConn struct {
	ReadBuf  *bytes.Buffer
	WriteBuf *bytes.Buffer
	Closed   bool
}

func NewMockNetConn() *MockNetConn {
	return &MockNetConn{
		ReadBuf:  &bytes.Buffer{},
		WriteBuf: &bytes.Buffer{},
	}
}

func (m *MockNetConn) Read(b []byte) (n int, err error) {
	return m.ReadBuf.Read(b)
}

func (m *MockNetConn) Write(b []byte) (n int, err error) {
	return m.WriteBuf.Write(b)
}

func (m *MockNetConn) Close() error {
	m.Closed = true
	return nil
}

func (m *MockNetConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (m *MockNetConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *MockNetConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockNetConn) SetWriteDeadline(t time.Time) error { return nil }
