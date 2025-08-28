package yamuxws

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSConn adapts a WebSocket connection to work with yamux as a net.Conn
type WSConn struct {
	conn       *websocket.Conn
	reader     io.Reader
	readMux    sync.Mutex
	writeMux   sync.Mutex
	localAddr  net.Addr
	remoteAddr net.Addr
}

// NewWSConn creates a new WebSocket connection adapter for yamux
func NewWSConn(conn *websocket.Conn) *WSConn {
	return &WSConn{
		conn:       conn,
		localAddr:  &wsAddr{conn.LocalAddr()},
		remoteAddr: &wsAddr{conn.RemoteAddr()},
	}
}

// Read implements io.Reader
func (c *WSConn) Read(b []byte) (int, error) {
	c.readMux.Lock()
	defer c.readMux.Unlock()

	if c.reader == nil {
		messageType, reader, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}
		if messageType != websocket.BinaryMessage {
			return 0, websocket.ErrReadLimit
		}
		c.reader = reader
	}

	n, err := c.reader.Read(b)
	if err == io.EOF {
		c.reader = nil
		return n, nil
	}
	return n, err
}

// Write implements io.Writer
func (c *WSConn) Write(b []byte) (int, error) {
	c.writeMux.Lock()
	defer c.writeMux.Unlock()

	err := c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes the WebSocket connection
func (c *WSConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address
func (c *WSConn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the remote network address  
func (c *WSConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline sets the read and write deadlines
func (c *WSConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

// SetReadDeadline sets the read deadline
func (c *WSConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline
func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// wsAddr implements net.Addr for WebSocket connections
type wsAddr struct {
	addr net.Addr
}

func (a *wsAddr) Network() string {
	if a.addr != nil {
		return a.addr.Network()
	}
	return "ws"
}

func (a *wsAddr) String() string {
	if a.addr != nil {
		return a.addr.String()
	}
	return "websocket"
}