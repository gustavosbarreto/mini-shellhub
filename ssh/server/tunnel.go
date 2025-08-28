package server

import (
	"fmt"
	"io"
	"net"
	"time"
)

// Tunnel interface for different tunnel implementations
type Tunnel interface {
	// Dial creates a connection to the specified target
	Dial(target string) (net.Conn, error)
}

// DeviceManager tunnel implementation
type DeviceManagerTunnel struct {
	deviceManager interface {
		OpenStream(deviceID string) (io.ReadWriteCloser, error)
	}
}

func NewDeviceManagerTunnel(dm interface {
	OpenStream(deviceID string) (io.ReadWriteCloser, error)
}) *DeviceManagerTunnel {
	return &DeviceManagerTunnel{deviceManager: dm}
}

func (t *DeviceManagerTunnel) Dial(target string) (net.Conn, error) {
	stream, err := t.deviceManager.OpenStream(target)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to device %s: %w", target, err)
	}
	
	// Convert stream to net.Conn
	return &streamConn{stream: stream, target: target}, nil
}

// streamConn adapts a stream to net.Conn
type streamConn struct {
	stream io.ReadWriteCloser
	target string
}

func (c *streamConn) Read(b []byte) (int, error) {
	return c.stream.Read(b)
}

func (c *streamConn) Write(b []byte) (int, error) {
	return c.stream.Write(b)
}

func (c *streamConn) Close() error {
	return c.stream.Close()
}

func (c *streamConn) LocalAddr() net.Addr {
	return &tunnelAddr{network: "yamux", address: "local"}
}

func (c *streamConn) RemoteAddr() net.Addr {
	return &tunnelAddr{network: "yamux", address: c.target}
}

func (c *streamConn) SetDeadline(_ time.Time) error {
	return nil // yamux handles deadlines internally
}

func (c *streamConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *streamConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type tunnelAddr struct {
	network, address string
}

func (a *tunnelAddr) Network() string {
	return a.network
}

func (a *tunnelAddr) String() string {
	return a.address
}