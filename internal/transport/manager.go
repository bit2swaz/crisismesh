package transport

import (
	"fmt"
	"net"
	"sync"
)

type Manager struct {
	conns sync.Map // map[string]net.Conn (key: remoteAddr)
}

func NewManager() *Manager {
	return &Manager{}
}

// Listen starts a TCP listener on the given port.
// It spawns a goroutine for each incoming connection that calls the handler.
func (m *Manager) Listen(port string, handler func(net.Conn)) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Log error but continue listening
				fmt.Printf("Accept error: %v\n", err)
				continue
			}

			m.registerConn(conn)
			go func(c net.Conn) {
				defer m.unregisterConn(c)
				defer c.Close()
				handler(c)
			}(conn)
		}
	}()

	return nil
}

// Dial connects to a remote address.
func (m *Manager) Dial(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	m.registerConn(conn)
	return conn, nil
}

func (m *Manager) registerConn(conn net.Conn) {
	m.conns.Store(conn.RemoteAddr().String(), conn)
}

func (m *Manager) unregisterConn(conn net.Conn) {
	m.conns.Delete(conn.RemoteAddr().String())
}

// CloseAll closes all active connections.
func (m *Manager) CloseAll() {
	m.conns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		return true
	})
}

// BroadcastPacket sends a data packet to all active connections.
func (m *Manager) BroadcastPacket(data []byte) {
	m.conns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			// We ignore errors here to ensure we try to send to everyone
			// In a real system, we might want to log this or remove dead connections
			_ = WriteFrame(conn, data)
		}
		return true
	})
}

// HasConnection checks if there is an active connection to the given address.
func (m *Manager) HasConnection(addr string) bool {
	_, ok := m.conns.Load(addr)
	return ok
}
