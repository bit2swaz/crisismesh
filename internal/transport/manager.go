package transport

import (
	"fmt"
	"net"
	"sync"
)

type Manager struct {
	conns sync.Map
}

func NewManager() *Manager {
	return &Manager{}
}
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
func (m *Manager) CloseAll() {
	m.conns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		return true
	})
}
func (m *Manager) BroadcastPacket(data []byte) {
	m.conns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			_ = WriteFrame(conn, data)
		}
		return true
	})
}
func (m *Manager) HasConnection(addr string) bool {
	_, ok := m.conns.Load(addr)
	return ok
}
func (m *Manager) SendPacket(addr string, data []byte) error {
	val, ok := m.conns.Load(addr)
	if !ok {
		return fmt.Errorf("no connection to %s", addr)
	}
	conn := val.(net.Conn)
	return WriteFrame(conn, data)
}
