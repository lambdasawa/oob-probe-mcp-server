package oob

import (
	"fmt"
	"net"

	"github.com/google/uuid"
)

type Port uint

type ListenerManager struct {
	tcpListeners  map[Port]*TCPListener
	httpListeners map[Port]*HTTPListener
}

type BackgroundError struct {
	Time    string `json:"time" jsonschema:"timestamp"`
	Message string `json:"message" jsonschema:"error message"`
}

func NewListenerManager() *ListenerManager {
	m := &ListenerManager{
		tcpListeners:  make(map[Port]*TCPListener),
		httpListeners: make(map[Port]*HTTPListener),
	}
	return m
}

func (m *ListenerManager) GetStatus() []map[string]any {
	items := make([]map[string]any, 0, len(m.tcpListeners)+len(m.httpListeners))
	for _, l := range m.tcpListeners {
		items = append(items, map[string]any{
			"id":         l.id,
			"protocol":   "tcp",
			"port":       l.backendPort,
			"public_url": l.tunnel.URL(),
			"connected":  l.conn != nil,
			"errors":     l.errors,
		})
	}
	for _, l := range m.httpListeners {
		items = append(items, map[string]any{
			"id":         l.id,
			"protocol":   "http",
			"port":       l.backendPort,
			"public_url": l.tunnel.URL(),
			"requests":   l.RequestCount(),
			"errors":     l.errors,
		})
	}
	return items
}

func (m *ListenerManager) ListenTCP() (*TCPListener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	actualPort := Port(ln.Addr().(*net.TCPAddr).Port)

	if _, exists := m.tcpListeners[actualPort]; exists {
		_ = ln.Close()
		return nil, fmt.Errorf("port %d already listening (tcp)", actualPort)
	}
	if _, exists := m.httpListeners[actualPort]; exists {
		_ = ln.Close()
		return nil, fmt.Errorf("port %d already listening (http)", actualPort)
	}

	l := &TCPListener{
		id:          uuid.New().String(),
		backendPort: actualPort,
		logStore:    new(LogStore),
		ln:          ln,
	}
	m.tcpListeners[actualPort] = l

	if err := l.StartNgrokTunnel(); err != nil {
		_ = l.Close()
		delete(m.tcpListeners, actualPort)
		return nil, fmt.Errorf("failed to start ngrok tunnel: %v", err)
	}

	go l.acceptLoop()

	return l, nil
}

func (m *ListenerManager) ListenHTTP(port Port) (*HTTPListener, error) {
	if _, exists := m.httpListeners[port]; exists {
		return nil, fmt.Errorf("port %d already listening (http)", port)
	}
	if _, exists := m.tcpListeners[port]; exists {
		return nil, fmt.Errorf("port %d already listening (tcp)", port)
	}

	l := &HTTPListener{
		id:          uuid.New().String(),
		backendPort: port,
		logStore:    new(LogStore),
	}
	m.httpListeners[port] = l

	if err := l.Start(); err != nil {
		_ = l.Close()
		delete(m.httpListeners, port)
		return nil, err
	}

	if err := l.StartNgrokTunnel(); err != nil {
		_ = l.Close()
		delete(m.httpListeners, port)
		return nil, fmt.Errorf("failed to start ngrok tunnel: %v", err)
	}

	return l, nil
}

func (m *ListenerManager) CloseTCP(port Port) error {
	l, exists := m.tcpListeners[port]
	if !exists {
		return fmt.Errorf("port %d not listening (tcp)", port)
	}
	delete(m.tcpListeners, port)
	return l.Close()
}

func (m *ListenerManager) CloseHTTP(port Port) error {
	l, exists := m.httpListeners[port]
	if !exists {
		return fmt.Errorf("port %d not listening (http)", port)
	}
	delete(m.httpListeners, port)
	return l.Close()
}

func (m *ListenerManager) CloseAll() {
	for _, l := range m.tcpListeners {
		_ = l.Close() // TODO: handle error
	}
	for _, l := range m.httpListeners {
		_ = l.Close() // TODO: handle error
	}

	m.tcpListeners = make(map[Port]*TCPListener)
	m.httpListeners = make(map[Port]*HTTPListener)
}

func (m *ListenerManager) GetTCP(port Port) (*TCPListener, bool) {
	l, ok := m.tcpListeners[port]
	return l, ok
}

func (m *ListenerManager) GetHTTP(port Port) (*HTTPListener, bool) {
	l, ok := m.httpListeners[port]
	return l, ok
}
